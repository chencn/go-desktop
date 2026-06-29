import { readdirSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(testDir, '..', '..')
const schemeEntryPath = resolve(repoRoot, 'frontend', 'src', 'styles', 'artistic-scheme.css')
const componentStyleDir = resolve(repoRoot, 'frontend', 'src', 'styles', 'artistic-scheme', 'components')

const usedComponentImports = [
  'alert-dialog',
  'badge',
  'button',
  'card',
  'dialog',
  'field',
  'input',
  'progress',
  'select',
  'switch',
  'table',
  'tooltip',
]

function artisticSchemeEntry() {
  return readFileSync(schemeEntryPath, 'utf8')
}

function componentStyleNames() {
  return readdirSync(componentStyleDir)
    .filter((fileName) => fileName.endsWith('.css'))
    .map((fileName) => fileName.replace(/\.css$/, ''))
    .sort()
}

function activeComponentImports(css: string) {
  return [...css.matchAll(/^@import "\.\/artistic-scheme\/components\/([^"]+)\.css";$/gm)]
    .map((match) => match[1])
    .sort()
}

describe('artistic scheme css entry', () => {
  it('only actively imports theme css for components used by the app', () => {
    expect(activeComponentImports(artisticSchemeEntry())).toEqual([...usedComponentImports].sort())
  })

  it('keeps unused component theme css discoverable as disabled imports with a reason', () => {
    const css = artisticSchemeEntry()
    const unusedComponentImports = componentStyleNames().filter((name) => !usedComponentImports.includes(name))

    for (const name of unusedComponentImports) {
      expect(css).toContain(`/* @import "./artistic-scheme/components/${name}.css"; -- 未启用：当前业务未引用该 primitive，避免打包未使用主题。 */`)
    }
  })
})
