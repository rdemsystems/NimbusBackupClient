// Checks src/i18n/translations.js against the English block, which is the
// reference. The UI has no fallback to English: a key missing from a language
// shows up as its raw name, so every language must carry every key.
//
// Run with `npm test` (no dependencies needed). Exits 1 on any problem.

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import translations from '../src/i18n/translations.js'

const src = (rel) => fileURLToPath(new URL(`../src/${rel}`, import.meta.url))
const errors = []
const placeholders = (s) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',')

// Nested blocks (featuresList.vss, ...) become dotted keys, as t() reads them.
const flatten = (obj, prefix = '') =>
  Object.entries(obj).flatMap(([k, v]) =>
    v !== null && typeof v === 'object' ? flatten(v, `${prefix}${k}.`) : [[`${prefix}${k}`, v]])

const ref = Object.fromEntries(flatten(translations.en))

for (const [lang, nested] of Object.entries(translations)) {
  const block = Object.fromEntries(flatten(nested))
  for (const key of Object.keys(ref)) {
    if (!(key in block)) errors.push(`${lang}: missing key "${key}"`)
  }
  for (const [key, value] of Object.entries(block)) {
    if (!(key in ref)) {
      errors.push(`${lang}: key "${key}" is not in en`)
    } else if (typeof value !== 'string') {
      errors.push(`${lang}: key "${key}" is not a string`)
    } else if (value.trim() === '' && ref[key] !== '') {
      // en leaves a few keys blank on purpose (basedOn); others must be filled
      errors.push(`${lang}: key "${key}" is empty`)
    } else if (placeholders(value) !== placeholders(ref[key])) {
      errors.push(`${lang}: key "${key}" placeholders {${placeholders(value)}} differ from en {${placeholders(ref[key])}}`)
    }
  }
}

// A key written twice in the same block is silently overwritten by the
// later one, so look for duplicates in the source text.
const text = readFileSync(src('i18n/translations.js'), 'utf8')
for (const [, lang, body] of text.matchAll(/^ {2}(\w+): \{\n([\s\S]*?)^ {2}\}/gm)) {
  const seen = new Set()
  for (const [, key] of body.matchAll(/^ {4,6}(\w+):/gm)) {
    if (seen.has(key)) errors.push(`${lang}: key "${key}" is defined twice`)
    seen.add(key)
  }
}

// Every language must be selectable in the switcher, and vice versa.
const switcher = readFileSync(src('components/LanguageSwitcher.jsx'), 'utf8')
const listed = new Set([...switcher.matchAll(/code: '(\w+)'/g)].map((m) => m[1]))
for (const lang of Object.keys(translations)) {
  if (!listed.has(lang)) errors.push(`${lang}: not listed in LanguageSwitcher.jsx`)
}
for (const lang of listed) {
  if (!(lang in translations)) errors.push(`${lang}: listed in LanguageSwitcher.jsx but has no translations`)
}

if (errors.length) {
  console.error(errors.join('\n'))
  console.error(`\n${errors.length} translation problem(s)`)
  process.exit(1)
}
console.log(`translations OK: ${Object.keys(translations).join(', ')} (${Object.keys(ref).length} keys each)`)
