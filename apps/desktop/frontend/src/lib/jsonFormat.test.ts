import { describe, it, expect } from 'vitest'
import { formatJsonWithPlaceholders } from './jsonFormat'

describe('formatJsonWithPlaceholders', () => {
  it('formats plain JSON', () => {
    expect(formatJsonWithPlaceholders('{"a":1,"b":[true]}')).toBe('{\n  "a": 1,\n  "b": [\n    true\n  ]\n}')
  })

  it('formats a body with bare placeholders for numbers, booleans, arrays and objects', () => {
    const input = '{"title":"{{title}}","price":{{price}},"inStock":{{inStock}},"tags":{{tags}},"size":{{size}}}'
    expect(formatJsonWithPlaceholders(input)).toBe(
      '{\n  "title": "{{title}}",\n  "price": {{price}},\n  "inStock": {{inStock}},\n  "tags": {{tags}},\n  "size": {{size}}\n}',
    )
  })

  it('leaves a placeholder inside a string alone, including braces and escapes around it', () => {
    expect(formatJsonWithPlaceholders('{"a":"x {{y}} \\" {{z}}","b":{{n}}}')).toBe('{\n  "a": "x {{y}} \\" {{z}}",\n  "b": {{n}}\n}')
  })

  it('keeps spaces inside a placeholder as typed', () => {
    expect(formatJsonWithPlaceholders('{"n":{{ count }}}')).toBe('{\n  "n": {{ count }}\n}')
  })

  it('returns null when it is not JSON even with the placeholders set aside', () => {
    expect(formatJsonWithPlaceholders('{"a": }')).toBeNull()
    expect(formatJsonWithPlaceholders('{"price": abc}')).toBeNull()
    expect(formatJsonWithPlaceholders('{{')).toBeNull()
    expect(formatJsonWithPlaceholders('')).toBeNull()
  })
})
