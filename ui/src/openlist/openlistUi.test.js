import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, it, expect } from 'vitest'

describe('OpenList UI wiring', () => {
  it('registers openlist resource before transcoding in App', () => {
    const appPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      '..',
      'App.jsx',
    )
    const source = fs.readFileSync(appPath, 'utf8')

    const openListPos = source.indexOf('name="openlist"')
    const transcodingPos = source.indexOf('name="transcoding"')

    expect(openListPos).toBeGreaterThan(-1)
    expect(transcodingPos).toBeGreaterThan(-1)
    expect(openListPos).toBeLessThan(transcodingPos)
  })

  it('defines password helper text in OpenList edit form', () => {
    const editPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      'OpenListEdit.jsx',
    )
    const source = fs.readFileSync(editPath, 'utf8')

    expect(source).toContain('source="openlistPass"')
    expect(source).toContain('resources.openlist.message.keepPassword')
    expect(source).toContain('source="enabled"')
  })

  it('does not append static record id to title', () => {
    const editPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      'OpenListEdit.jsx',
    )
    const source = fs.readFileSync(editPath, 'utf8')

    expect(source).not.toContain('record ? record.id')
  })

  it('disables default edit actions to hide redundant delete button', () => {
    const editPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      'OpenListEdit.jsx',
    )
    const source = fs.readFileSync(editPath, 'utf8')

    expect(source).toContain('actions={false}')
    expect(source).toContain('toolbar={<OpenListToolbar />}')
    expect(source).not.toContain('DeleteButton')
  })
})
