// Pretty-prints a JSON request body that may contain {{placeholders}}. A placeholder inside a string
// ("{{name}}") is just text; a bare one ({{price}}, used for numbers, booleans, arrays and objects)
// is not valid JSON until it is filled in, so it is swapped for a marker while formatting and put
// back afterwards. Returns null when the text is not JSON even with the placeholders set aside.
export function formatJsonWithPlaceholders(text: string): string | null {
  const saved: string[] = []
  let masked = ''
  let inString = false
  for (let i = 0; i < text.length; i++) {
    const c = text[i]
    if (inString) {
      masked += c
      if (c === '\\') masked += text[++i] ?? ''
      else if (c === '"') inString = false
      continue
    }
    if (c === '"') {
      inString = true
      masked += c
      continue
    }
    if (c === '{' && text[i + 1] === '{') {
      const end = text.indexOf('}}', i + 2)
      if (end > 0) {
        saved.push(text.slice(i, end + 2))
        masked += `"__CS_PLACEHOLDER_${saved.length - 1}__"`
        i = end + 1
        continue
      }
    }
    masked += c
  }
  try {
    const formatted = JSON.stringify(JSON.parse(masked), null, 2)
    return formatted.replace(/"__CS_PLACEHOLDER_(\d+)__"/g, (_, n: string) => saved[Number(n)] ?? '')
  } catch {
    return null
  }
}
