import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'

const root = process.cwd()
const srcDir = join(root, 'src')

const checks = [
  {
    name: 'Hardcoded hex color',
    regex: /#[0-9a-fA-F]{3,8}/g,
    message: 'Use nesio design tokens instead of hardcoded hex colors.',
  },
  {
    name: 'Arbitrary text size',
    regex: /text-\[[^\]]+\]/g,
    message: 'Use type-* classes instead of text-[...].',
  },
  {
    name: 'Arbitrary radius',
    regex: /rounded-\[[^\]]+\]/g,
    message: 'Use radius scale classes instead of rounded-[...].',
  },
  {
    name: 'Framework palette color',
    regex: /(bg|text|border)-(red|blue|green|yellow|orange|slate|gray)-\d{2,3}/g,
    message: 'Map semantic colors to nesio tokens.',
  },
]

function walk(dir) {
  const entries = readdirSync(dir)
  const files = []
  for (const entry of entries) {
    const fullPath = join(dir, entry)
    const stats = statSync(fullPath)
    if (stats.isDirectory()) {
      files.push(...walk(fullPath))
      continue
    }
    if (fullPath.endsWith('.tsx')) {
      files.push(fullPath)
    }
  }
  return files
}

const files = walk(srcDir)
const failures = []

for (const file of files) {
  const content = readFileSync(file, 'utf8')
  const lines = content.split('\n')

  lines.forEach((line, index) => {
    checks.forEach((check) => {
      check.regex.lastIndex = 0
      if (check.regex.test(line)) {
        failures.push({
          file,
          line: index + 1,
          type: check.name,
          detail: check.message,
          snippet: line.trim(),
        })
      }
    })
  })
}

if (failures.length > 0) {
  console.error('\nUI style guard failed.\n')
  failures.forEach((item) => {
    console.error(`- ${item.file}:${item.line}`)
    console.error(`  [${item.type}] ${item.detail}`)
    console.error(`  ${item.snippet}`)
  })
  console.error(`\nTotal violations: ${failures.length}\n`)
  process.exit(1)
}

console.log('UI style guard passed.')
