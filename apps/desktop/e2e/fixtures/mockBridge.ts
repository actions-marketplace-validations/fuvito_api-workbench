/**
 * Injected into the page via page.addInitScript before React loads.
 * Sets up window.go (Wails IPC) and window.runtime stubs so the app
 * renders without a live Go backend.
 *
 * Tests can override specific behaviors after page.goto() via page.evaluate:
 *   await page.evaluate(() => {
 *     (window as any).__bridge.executeResult = { statusCode: 404, ... }
 *   })
 */
export function setupMockBridge() {
  const noop = () => Promise.resolve(undefined)
  const noopNull = () => Promise.resolve(null)

  const DEFAULT_SETTINGS = {
    theme: 'dark',
    fontSize: 13,
    defaultTimeoutMs: 30000,
    proxyUrl: '',
    tlsSkipVerify: false,
  }

  const DEFAULT_RESPONSE = {
    statusCode: 200,
    status: 'OK',
    body: '{"ok":true}',
    headers: [{ name: 'Content-Type', value: 'application/json' }],
    durationMs: 123,
    size: 11,
    error: '',
    timing: { dnsMs: 1, connectMs: 5, tlsMs: 0, ttfbMs: 100, transferMs: 17 },
  }

  // Mutable bridge state: tests can overwrite fields after goto()
  ;(window as unknown as Record<string, unknown>)['__bridge'] = {
    settings: { ...DEFAULT_SETTINGS },
    executeResult: { ...DEFAULT_RESPONSE },
    history: [] as unknown[],
    workspacePath: '/workspace',
    fileTree: [] as unknown[],
    environments: [] as unknown[],
    activeEnv: null,
    scenarios: [] as unknown[],
    scenarioDetail: null as unknown,
    scenarioResult: null as unknown,
    scenarioRaw: '' as unknown,
    scenarioResultCache: {} as Record<string, unknown>,
    recentWorkspaces: [] as unknown[],
    coverage: [] as Array<{ specFile: string }>,
    pickedFile: '' as string,
    cliCatalog: [
      {
        name: 'run', goal: 'Run scenarios (a file or a whole folder)', summary: 'Runs scenario files step by step: sends each request, extracts values, and checks every assertion. Give it one file, several, or a folder to run everything inside it.', when: 'To check a workflow such as login, create, fetch, delete, or a whole suite, locally or as a CI gate.',
        inputs: [
          { name: 'scenario-file', short: '', positional: true, required: true, kind: 'path', role: 'scenario', accepts: ['.yaml'], enum: [], default: '', label: 'Scenario or folder', help: 'A scenario file, or a folder to run every scenario inside it (and in its sub-folders). With several scenarios you get one combined report and one exit code.', example: 'scenarios/user-lifecycle.yaml', group: 'essential' },
          { name: 'env', short: 'e', positional: false, required: false, kind: 'file', role: 'environment', accepts: ['.yaml'], enum: [], default: '', label: 'Environment', help: 'Variables used by placeholders in your requests, such as baseUrl and tokens.', example: 'environments/ci.yaml', group: 'essential' },
          { name: 'coverage', short: '', positional: false, required: false, kind: 'bool', role: '', accepts: [], enum: [], default: 'false', label: 'Coverage report', help: 'After the run, list which operations of the spec the scenario exercised.', example: '', group: 'essential' },
          { name: 'format', short: 'f', positional: false, required: false, kind: 'enum', role: '', accepts: [], enum: ['text', 'json', 'junit', 'html'], default: 'text', label: 'Report format', help: 'How the report is written. text is for people reading a terminal or CI log; json is for scripts; junit is for CI test dashboards; html is a single page you can open in a browser or keep as a build artifact.', example: '', group: 'essential' },
          { name: 'output', short: 'o', positional: false, required: false, kind: 'outputFile', role: '', accepts: [], enum: [], default: '', label: 'Save report to', help: 'Write the report to this file instead of the screen. Missing folders are created. Nothing is written to disk unless you set this.', example: 'reports/run.html', group: 'essential' },
        ],
        outputs: ['A report of every step and assertion, on the screen.', 'With a workspace: coverage recorded for the Contracts page.'],
        exitCodes: [
          { code: 0, meaning: 'Every step passed every assertion.', forCI: 'The pipeline step succeeds.' },
          { code: 1, meaning: 'At least one assertion failed.', forCI: 'The pipeline step fails: the API no longer behaves as the scenario expects.' },
          { code: 2, meaning: 'The run could not happen: a file could not be read or parsed.', forCI: 'The pipeline step fails: fix the command or the files first.' },
        ],
      },
      {
        name: 'conform', goal: 'Check an API against its OpenAPI spec', summary: 'Calls every operation in the spec and checks each response against what the spec declares.', when: 'To confirm a running API still matches its specification, locally or as a CI gate.',
        inputs: [
          { name: 'spec', short: '', positional: true, required: true, kind: 'path', role: 'spec', accepts: ['.yaml'], enum: [], default: '', label: 'OpenAPI spec', help: 'The spec file to check against.', example: 'specs/openapi.yaml', group: 'essential' },
          { name: 'base-url', short: 'b', positional: false, required: false, kind: 'text', role: '', accepts: [], enum: [], default: '', label: 'Base URL', help: 'Where the API runs.', example: 'http://localhost:8120', group: 'essential' },
        ],
        outputs: [], exitCodes: [],
      },
    ] as unknown[],
    cliContext: {
      workspace: '/workspace', runFrom: '/workspace',
      scenarios: [{ path: '/workspace/scenarios/user-lifecycle.yaml', label: 'scenarios/user-lifecycle.yaml' }],
      scenarioFolders: [{ path: '/workspace/scenarios', label: 'scenarios/ — 1 scenario' }],
      environments: [{ path: '/workspace/environments/ci.yaml', label: 'environments/ci.yaml' }],
      specs: [],
    } as unknown,
  }

  const bridge =(window as unknown as Record<string, Record<string, unknown>>)['__bridge']

  ;(window as unknown as Record<string, unknown>)['go'] = {
    main: {
      App: {
        SetHasDirtyTabs: noop,
      },
      CliBuilderService: {
        GetCliCatalog: () => Promise.resolve(bridge['cliCatalog']),
        GetCliContext: () => Promise.resolve(bridge['cliContext']),
        PickCliFile: () => Promise.resolve(bridge['pickedFile'] ?? ''),
        PickCliFolder: () => Promise.resolve(''),
        BuildCliCommand: (input: { command: string; values: Record<string, string>; runFrom: string }) => {
          type Inp = { name: string; positional: boolean; required: boolean; kind: string; label: string }
          const cmd = (bridge['cliCatalog'] as Array<{ name: string; inputs: Inp[]; exitCodes: unknown[]; outputs: string[]; summary: string; goal: string; when: string }>)
            .find(c => c.name === input.command)
          const rel = (v: string) => (v.startsWith(input.runFrom + '/') ? v.slice(input.runFrom.length + 1) : v)
          const messages: Array<{ field: string; severity: string; text: string }> = []
          const positional: string[] = []
          const flags: string[] = []
          for (const inp of cmd?.inputs ?? []) {
            const v = input.values[inp.name]
            if (!v) {
              if (inp.required) messages.push({ field: inp.name, severity: 'error', text: `${inp.label} is required` })
              continue
            }
            if (inp.positional) positional.push(rel(v))
            else if (inp.kind === 'bool') flags.push(`--${inp.name}`)
            else if (inp.name === 'format' && v === 'text') continue
            else flags.push(`--${inp.name} ${rel(v)}`)
          }
          const fmt = input.values['format']
          const suggestions: Array<{ field: string; value: string; why: string }> = []
          if (fmt && fmt !== 'text' && !input.values['output']) {
            const ext = fmt === 'junit' ? 'xml' : fmt
            const subject = (input.values['scenario-file'] ?? '').split('/').pop()?.replace(/\.[a-z]+$/, '')
            suggestions.push({ field: 'output', value: `reports/${input.command}${subject ? '-' + subject : ''}.${ext}`, why: 'keeps reports together, named after the command and what it checks' })
            messages.push({ field: 'output', severity: 'warning', text: `without a file, the whole ${fmt.toUpperCase()} report is printed to the screen; choose where to save it` })
          }
          const target = input.values['scenario-file']
          if (target) {
            const isFolder = !/[.](ya?ml|json)$/.test(target)
            messages.push({
              field: 'scenario-file', severity: 'info',
              text: isFolder
                ? '3 scenarios found in ' + rel(target) + ' and its sub-folders, all valid; 1 other file skipped (not scenarios)'
                : 'valid, 4 steps',
            })
          }
          const first = 'clrsig ' + [input.command, ...positional].join(' ')
          const lines = [first, ...flags]
          const NL = String.fromCharCode(10)
          const BS = String.fromCharCode(92)
          const BT = String.fromCharCode(96)
          const join = (cont: string) => lines.join(' ' + cont + NL + '  ')
          const one = [first, ...flags].join(' ')
          return Promise.resolve({
            command: input.command, valid: !messages.some(m => m.severity === 'error'), argv: one.split(' '),
            renderings: [
              { shell: 'bash', multiline: join(BS), oneline: one },
              { shell: 'powershell', multiline: join(BT), oneline: one },
              { shell: 'cmd', multiline: join('^').split('/').join(BS), oneline: one.split('/').join(BS) },
            ],
            messages,
            suggestions,
            needs: input.values['env'] ? [{ name: 'API_BASE_URL', secret: false, from: 'baseUrl' }, { name: 'API_TOKEN', secret: true, from: 'token' }] : [],
            explanation: {
              goal: cmd?.goal ?? '', summary: cmd?.summary ?? '', when: cmd?.when ?? '',
              parts: [
                { token: 'clrsig', meaning: 'The Clear Signal command-line program.' },
                { token: input.command, meaning: cmd?.summary ?? '' },
                ...positional.map(p => ({ token: p, meaning: 'Scenario: The scenario file to run. It lists the steps, requests and assertions.' })),
              ],
              outputs: cmd?.outputs ?? [], exitCodes: cmd?.exitCodes ?? [],
            },
          })
        },
      },
      RequestService: {
        Execute: () => Promise.resolve(bridge['executeResult']),
        CancelRequest: noop,
        GetHistory: () => Promise.resolve(bridge['history']),
        ClearHistory: () => { bridge['history'] = []; return Promise.resolve() },
        RebuildExecutor: noop,
      },
      WorkspaceService: {
        GetWorkspacePath: () => Promise.resolve(bridge['workspacePath']),
        ListRequests: () => Promise.resolve(bridge['fileTree']),
        ListEnvironments: () => Promise.resolve(bridge['environments']),
        GetActiveEnvironment: () => Promise.resolve(bridge['activeEnv']),
        GetEnvironment: () => Promise.resolve({ name: '', variables: [] }),
        ActiveEnvironment: noopNull,
        OpenWorkspace: () => { bridge['workspacePath'] = '/workspace'; return Promise.resolve('/workspace') },
        NewWorkspace: () => { bridge['workspacePath'] = '/workspace'; return Promise.resolve('/workspace') },
        CloseWorkspace: () => { bridge['workspacePath'] = ''; return Promise.resolve() },
        OpenWorkspaceByPath: (path: string) => { bridge['workspacePath'] = path; return Promise.resolve(path) },
        GetRecentWorkspaces: () => Promise.resolve(bridge['recentWorkspaces'] ?? []),
        SaveRequest: (r: Record<string, string>) =>
          Promise.resolve('/workspace/' + (r?.name || 'req') + '.yaml'),
        LoadRequest: noopNull,
        OpenRequest: noopNull,
        DeleteRequest: noop,
        CreateFolder: noop,
        DeleteFolder: noop,
        RenameRequest: (p: string) => Promise.resolve(p),
        RenameFolder: (p: string) => Promise.resolve(p),
        MoveRequest: (src: string) => Promise.resolve(src),
        SaveResponseCache: noop,
        LoadResponseCache: noopNull,
        ClearResponseCache: noop,
        PickFile: noopNull,
        SaveContentToFile: () => Promise.resolve(''),
        OpenInExplorer: () => Promise.resolve(''),
        OpenInSystemViewer: noop,
        SaveEnvironment: (e: Record<string, string>) =>
          Promise.resolve('/workspace/environments/' + (e?.name || 'env') + '.yaml'),
        DeleteEnvironment: noop,
        SelectEnvironment: noop,
      },
      SettingsService: {
        GetSettings: () => Promise.resolve({ ...(bridge['settings'] as object) }),
        SaveSettings: (s: unknown) => { bridge['settings'] = s as Record<string, unknown>; return Promise.resolve(s) },
        ResetSettings: () => {
          bridge['settings'] = { ...DEFAULT_SETTINGS }
          return Promise.resolve({ ...DEFAULT_SETTINGS })
        },
      },
      DemoService: {
        StartDemo: () => Promise.resolve({ port: 8080, error: '' }),
        StopDemo: noop,
        IsRunning: () => Promise.resolve(false),
        GetPort: () => Promise.resolve(0),
      },
      ConformanceService: {
        GetConformContext: () => Promise.resolve({
          workspace: '/workspace',
          specs: [{ path: '/workspace/specs/shop.yaml', label: 'specs/shop.yaml' }],
          environments: [{ path: '/workspace/environments/local.yaml', label: 'environments/local.yaml' }],
        }),
        InspectConformSpec: () => Promise.resolve({ title: 'Shop', version: '1', serverUrl: 'http://localhost:8120', reads: 3, writes: 2 }),
        RunConformance: (input: { baseUrl: string; allowWrites: boolean; confirmedRemote: boolean; negative: boolean; tags: string[] }) => {
          bridge['conformRuns'] = [...((bridge['conformRuns'] as unknown[]) ?? []), input]
          const local = /^https?:\/\/(localhost|127\.0\.0\.1)(:|\/|$)/.test(input.baseUrl)
          const empty = { specTitle: '', baseUrl: input.baseUrl, writes: false, durationMs: 0, operations: [], total: 0, covered: 0, notCovered: 0, passing: 0, failing: 0, probes: [], warnings: [] }
          if (!local && !input.confirmedRemote) return Promise.resolve({ ...empty, error: 'confirm the base URL ' + input.baseUrl + ' before running: it is not on this computer' })
          const op = (operationId: string, method: string, path: string, status: string, statusCode: number, violations: string[] = []) =>
            ({ operationId, method, path, status, statusCode, durationMs: 4, violations })
          const operations = [
            op('listItems', 'GET', '/items', 'passed', 200),
            op('getItem', 'GET', '/items/{{id}}', 'failed', 200, ['Error at "/price": value must be a number']),
            op('health', 'GET', '/health', 'passed', 200),
            input.allowWrites ? op('createItem', 'POST', '/items', 'passed', 201) : op('createItem', 'POST', '/items', 'skipped', 0),
            input.allowWrites ? op('deleteItem', 'DELETE', '/items/{{id}}', 'passed', 204) : op('deleteItem', 'DELETE', '/items/{{id}}', 'skipped', 0),
          ]
          const covered = operations.filter(o => o.status === 'passed' || o.status === 'failed').length
          const probes = input.negative
            ? [{ operationId: 'getItem', method: 'GET', path: '/items/{{id}}', kind: 'wrong-type', description: 'id as text', status: 'accepted', statusCode: 200 }]
            : []
          return Promise.resolve({
            ...empty, specTitle: 'Shop', writes: input.allowWrites, durationMs: 21, operations, total: 5, covered,
            notCovered: 5 - covered, passing: operations.filter(o => o.status === 'passed').length, failing: 1, probes,
          })
        },
        CancelConformance: noop,
        ExportConformReport: () => Promise.resolve({ content: '', error: '' }),
      },
      ConformanceServerService: {
        Start: () => Promise.resolve({ port: 8120, error: '' }),
        OpenWorkspace: () => Promise.resolve({ port: 8120, workspacePath: '', error: '' }),
        Stop: noop,
        IsRunning: () => Promise.resolve(false),
        GetPort: () => Promise.resolve(0),
        OpenSpecInBrowser: () => Promise.resolve(''),
      },
      OpenAPIService: {
        ImportOpenAPISpec: () => Promise.resolve({ count: 0, error: '' }),
        ImportOpenAPISpecFromPath: () => Promise.resolve({ count: 0, error: '' }),
        ImportOpenAPISpecFromURL: () => Promise.resolve({ count: 0, error: '' }),
      },
      ImportService: {
        ImportPostmanCollection: () => Promise.resolve({ count: 0, error: '' }),
        ImportInsomniaExport: () => Promise.resolve({ count: 0, error: '' }),
      },
      ScenarioService: {
        ListScenarios: () => Promise.resolve(bridge['scenarios']),
        ListTemplates: () => Promise.resolve([]),
        RunFolder: () => Promise.resolve([]),
        CreateFromTemplate: (templateId: string) => {
          const path = `/workspace/scenarios/${templateId}.yaml`
          return Promise.resolve({ path, error: '' })
        },
        LoadScenario: () => Promise.resolve(bridge['scenarioDetail'] ?? { path: '', title: '', description: '', error: '', steps: [] }),
        RunScenario: (_path: string, _envPath: string) => Promise.resolve(bridge['scenarioResult'] ?? { title: '', error: '', passed: true, durationMs: 0, steps: [] }),
        ListAccumulatedCoverage: () => Promise.resolve(bridge['coverage']),
        ListDynamicFunctions: () => Promise.resolve([
          { name: 'guid', usage: '$guid', description: 'Random UUID (version 4)', example: '$guid' },
          { name: 'randomInt', usage: '$randomInt [<min> <max>]', description: 'Random whole number (default 0 to 1000)', example: '$randomInt 1 100' },
        ]),
        GetAccumulatedCoverage: (specFile: string) =>
          Promise.resolve((bridge['coverage'] as Array<{ specFile: string }>).find(c => c.specFile === specFile) ?? null),
        ResetAccumulatedCoverage: (specFile: string) => {
          bridge['coverage'] = (bridge['coverage'] as Array<{ specFile: string }>).filter(c => c.specFile !== specFile)
          return Promise.resolve()
        },
        LoadScenarioRaw: () => Promise.resolve(bridge['scenarioRaw'] ?? ''),
        SaveScenarioRaw: (_path: string, content: string) => { bridge['scenarioRaw'] = content; return Promise.resolve('') },
        SaveScenarioResult: (path: string, result: unknown) => {
          (bridge['scenarioResultCache'] as Record<string, unknown>)[path] = result
          return Promise.resolve('')
        },
        LoadScenarioResult: (path: string) => {
          const cached = (bridge['scenarioResultCache'] as Record<string, unknown>)[path]
          return Promise.resolve(cached ?? null)
        },
        CreateScenario: (name: string) => {
          const slug = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
          const path = `/workspace/scenarios/${slug || 'scenario'}.yaml`
          const newScenario = { title: name, path, stepCount: 1 }
          ;(bridge['scenarios'] as unknown[]).push(newScenario)
          bridge['scenarioDetail'] = { path, title: name, description: '', steps: [], error: '' }
          bridge['scenarioRaw'] = `schemaVersion: "1"\ntitle: ${name}\nsteps:\n  - label: Step 1\n`
          return Promise.resolve({ path, error: '' })
        },
        DuplicateScenario: (path: string) => {
          const scenarios = bridge['scenarios'] as Array<{path: string; title: string; stepCount?: number}>
          const src = scenarios.find(s => s.path === path)
          if (!src) return Promise.resolve({ path: '', error: 'not found' })
          const dir = path.substring(0, path.lastIndexOf('/'))
          const base = path.substring(path.lastIndexOf('/') + 1).replace(/\.(yaml|yml|json)$/i, '')
          const newPath = `${dir}/${base}-copy.yaml`
          const newTitle = (src.title ?? '') + ' Copy'
          scenarios.push({ path: newPath, title: newTitle, stepCount: src.stepCount ?? 0 })
          return Promise.resolve({ path: newPath, error: '' })
        },
        DeleteScenario: (path: string) => {
          bridge['scenarios'] = (bridge['scenarios'] as Array<{path: string}>).filter(s => s.path !== path)
          return Promise.resolve('')
        },
        RenameScenario: (path: string, newName: string) => {
          const slug = newName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
          const dir = path.substring(0, path.lastIndexOf('/'))
          const newPath = `${dir}/${slug || 'scenario'}.yaml`
          bridge['scenarios'] = (bridge['scenarios'] as Array<{path: string; title: string}>).map(
            s => s.path === path ? { ...s, path: newPath, title: newName } : s
          )
          return Promise.resolve(newPath)
        },
      },
    },
  }

  // Wails runtime stubs (window.runtime is used by wailsjs/runtime/runtime.js)
  ;(window as unknown as Record<string, unknown>)['runtime'] = {
    LogPrint: noop, LogTrace: noop, LogDebug: noop,
    LogInfo: noop, LogWarning: noop, LogError: noop,
    EventsOn: () => {}, EventsOff: () => {}, EventsOnce: () => {}, EventsEmit: () => {},
    WindowMinimise: () => {}, WindowMaximise: () => {}, WindowUnmaximise: () => {},
    WindowIsMaximised: () => Promise.resolve(false),
    WindowClose: () => {}, WindowSetTitle: () => {},
    BrowserOpenURL: () => {},
  }
}
