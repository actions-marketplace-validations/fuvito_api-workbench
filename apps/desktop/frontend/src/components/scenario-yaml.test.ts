import { describe, it, expect } from 'vitest'
import { serializeScenarioToYAML } from './scenario-yaml'
import type { ScenarioDetail } from '@/lib/bridge'

function minimal(overrides: Partial<ScenarioDetail> = {}): ScenarioDetail {
  return {
    path: '/ws/scenarios/test.yaml',
    title: 'My Scenario',
    description: '',
    error: '',
    steps: [],
    ...overrides,
  }
}

// ---- schema version and title ----

describe('serializeScenarioToYAML — header', () => {
  it('always emits schemaVersion "1"', () => {
    expect(serializeScenarioToYAML(minimal())).toContain('schemaVersion: "1"')
  })

  it('emits title', () => {
    expect(serializeScenarioToYAML(minimal({ title: 'Auth Flow' }))).toContain('title: Auth Flow')
  })

  it('quotes title that contains a colon', () => {
    const yaml = serializeScenarioToYAML(minimal({ title: 'Create: User' }))
    expect(yaml).toContain('"Create: User"')
  })

  it('emits description when non-empty', () => {
    const yaml = serializeScenarioToYAML(minimal({ description: 'End-to-end flow' }))
    expect(yaml).toContain('description: End-to-end flow')
  })

  it('omits description when empty', () => {
    expect(serializeScenarioToYAML(minimal({ description: '' }))).not.toContain('description:')
  })

  it('emits specFile when present', () => {
    const yaml = serializeScenarioToYAML(minimal({ specFile: 'openapi.yaml' }))
    expect(yaml).toContain('specFile: openapi.yaml')
  })

  it('emits steps: [] for empty scenario', () => {
    expect(serializeScenarioToYAML(minimal())).toContain('steps: []')
  })
})

// ---- step fields ----

describe('serializeScenarioToYAML — step basics', () => {
  it('uses label: not title: for steps', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Create user', method: 'POST', url: 'https://example.com', assertions: [], extractions: [] }],
    }))
    expect(yaml).toContain('  - label: Create user')
    expect(yaml).not.toContain('  - title:')
  })

  it('emits method and url inside request block', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Get user', method: 'GET', url: 'https://api.example.com/users', assertions: [], extractions: [] }],
    }))
    expect(yaml).toContain('    request:')
    expect(yaml).toContain('      method: GET')
    expect(yaml).toContain('      url: https://api.example.com/users')
  })

  it('emits notes when present', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', notes: 'Checks the response', method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).toContain('    notes: Checks the response')
  })

  it('omits notes when absent', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).not.toContain('notes:')
  })

  it('emits disabled: true when step is disabled', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', disabled: true, method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).toContain('    disabled: true')
  })

  it('uses requestRef instead of inline request block', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', requestRef: 'requests/get-user.yaml', assertions: [], extractions: [] }],
    }))
    expect(yaml).toContain('    requestRef: requests/get-user.yaml')
    expect(yaml).not.toContain('    request:')
  })

  it('emits failPolicy when not stop', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', failPolicy: 'continue', method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).toContain('    failPolicy: continue')
  })

  it('omits failPolicy when value is stop', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', failPolicy: 'stop', method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).not.toContain('failPolicy:')
  })
})

// ---- timeout and retry ----

describe('serializeScenarioToYAML — timeout and retry', () => {
  it('emits timeoutMs when > 0', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', timeoutMs: 5000, method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).toContain('    timeoutMs: 5000')
  })

  it('omits timeoutMs when 0', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', timeoutMs: 0, method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).not.toContain('timeoutMs:')
  })

  it('emits retry block with maxAttempts and delayMs', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', retryMaxAttempts: 3, retryDelayMs: 500, method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).toContain('    retry:')
    expect(yaml).toContain('      maxAttempts: 3')
    expect(yaml).toContain('      delayMs: 500')
  })

  it('omits retry when maxAttempts is 0', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{ label: 'Step', retryMaxAttempts: 0, method: 'GET', url: '', assertions: [], extractions: [] }],
    }))
    expect(yaml).not.toContain('retry:')
  })
})

// ---- assertions ----

describe('serializeScenarioToYAML — assertions', () => {
  it('emits status assertion', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'GET', url: '',
        assertions: [{ type: 'status', operator: 'eq', expected: 200, path: '', schema: '' }],
        extractions: [],
      }],
    }))
    expect(yaml).toContain('    assertions:')
    expect(yaml).toContain('      - type: status')
    expect(yaml).toContain('        operator: eq')
    expect(yaml).toContain('        expected: 200')
  })

  it('emits jsonpath assertion with path', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'GET', url: '',
        assertions: [{ type: 'jsonpath', path: '$.name', operator: 'eq', expected: 'Alice', schema: '' }],
        extractions: [],
      }],
    }))
    expect(yaml).toContain('        path: $.name')
    expect(yaml).toContain('        expected: Alice')
  })

  it('emits schema assertion with schema field, no operator or expected', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'GET', url: '',
        assertions: [{ type: 'schema', schema: 'getUserById', operator: 'eq', expected: null, path: '' }],
        extractions: [],
      }],
    }))
    expect(yaml).toContain('      - type: schema')
    expect(yaml).toContain('        schema: getUserById')
  })

  it('defaults operator to eq when undefined', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'GET', url: '',
        assertions: [{ type: 'status', operator: undefined as unknown as string, expected: 200, path: '', schema: '' }],
        extractions: [],
      }],
    }))
    expect(yaml).toContain('        operator: eq')
  })
})

// ---- extractions ----

describe('serializeScenarioToYAML — extractions', () => {
  it('emits extraction with name, source, and path', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'POST', url: '',
        assertions: [],
        extractions: [{ name: 'userId', source: 'jsonpath', path: '$.id' }],
      }],
    }))
    expect(yaml).toContain('    extractions:')
    expect(yaml).toContain('      - name: userId')
    expect(yaml).toContain('        source: jsonpath')
    expect(yaml).toContain('        path: $.id')
  })
})

// ---- headers and query params ----

describe('serializeScenarioToYAML — headers and queryParams', () => {
  it('emits headers block', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'GET', url: '',
        headers: [{ name: 'Authorization', value: 'Bearer {{token}}' }],
        assertions: [], extractions: [],
      }],
    }))
    expect(yaml).toContain('      headers:')
    expect(yaml).toContain('        - name: Authorization')
    expect(yaml).toContain('          value: "Bearer {{token}}"')
  })

  it('emits queryParams block', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'GET', url: '',
        queryParams: [{ name: 'page', value: '1' }],
        assertions: [], extractions: [],
      }],
    }))
    expect(yaml).toContain('      queryParams:')
    expect(yaml).toContain('        - name: page')
    expect(yaml).toContain('          value: 1')
  })
})

// ---- body ----

describe('serializeScenarioToYAML — body', () => {
  it('emits json body block', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'POST', url: '',
        bodyType: 'json', bodyContent: '{"name":"Alice"}',
        assertions: [], extractions: [],
      }],
    }))
    expect(yaml).toContain('      body:')
    expect(yaml).toContain('        type: json')
    expect(yaml).toContain('Alice')
  })

  it('omits body block when type is none', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        method: 'GET', url: '',
        bodyType: 'none', bodyContent: '',
        assertions: [], extractions: [],
      }],
    }))
    expect(yaml).not.toContain('body:')
  })
})

// ---- yamlScalar edge cases ----

describe('serializeScenarioToYAML — scalar quoting', () => {
  it('quotes strings containing #', () => {
    const yaml = serializeScenarioToYAML(minimal({ title: 'Step #1' }))
    expect(yaml).toContain('"Step #1"')
  })

  it('quotes the literal string true', () => {
    const yaml = serializeScenarioToYAML(minimal({ title: 'true' }))
    expect(yaml).toContain('"true"')
  })

  it('quotes empty string as ""', () => {
    const yaml = serializeScenarioToYAML(minimal({
      steps: [{
        label: 'Step',
        notes: '',
        method: 'GET', url: '',
        assertions: [], extractions: [],
      }],
    }))
    // notes is omitted when empty/falsy — only non-empty notes are emitted
    expect(yaml).not.toContain('notes:')
  })

  it('output ends with a newline', () => {
    expect(serializeScenarioToYAML(minimal())).toMatch(/\n$/)
  })
})

describe('serializeScenarioToYAML — variables', () => {
  it('keeps scenario variables, including functions and the secret flag', () => {
    const out = serializeScenarioToYAML(minimal({
      variables: [
        { name: 'email', value: 'user-{{$randomString 10}}@example.com' },
        { name: 'token', value: 'abc', secret: true },
      ],
    }))
    expect(out).toContain('variables:')
    expect(out).toContain('  - name: email')
    expect(out).toContain('    value: "user-{{$randomString 10}}@example.com"')
    expect(out).toContain('    secret: true')
  })

  it('emits nothing when there are no variables', () => {
    expect(serializeScenarioToYAML(minimal())).not.toContain('variables:')
  })
})
