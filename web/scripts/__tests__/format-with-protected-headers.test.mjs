/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import {
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { after, describe, test } from 'node:test'
import { fileURLToPath } from 'node:url'

// Every fixture lives under a fresh directory in tmpdir(); the real web tree is
// never the working directory of the script under test. Set KEEP_TEST_DIR=1 to
// keep the fixtures as evidence.
const script = fileURLToPath(
  new URL('../format-with-protected-headers.mjs', import.meta.url)
)
const base = mkdtempSync(join(tmpdir(), 'protected-format-test-'))
after(() => {
  if (!process.env.KEEP_TEST_DIR) rmSync(base, { recursive: true, force: true })
})

// Fake formatter, invoked as: node fake.mjs --write <files...> (cwd = copy dir).
const fakeSource = [
  "import { readFileSync, rmSync, writeFileSync } from 'node:fs'",
  'const e = process.env',
  'const files = process.argv.slice(3)',
  "if (e.FAKE_FAIL) { console.error('fake failure'); process.exit(3) }",
  'if (e.FAKE_HANG) await new Promise((r) => setTimeout(r, 30000))',
  "if (e.FAKE_EDIT_SOURCE) writeFileSync(e.FAKE_EDIT_SOURCE, 'edited while formatting')",
  "for (const f of files) writeFileSync(f, readFileSync(f, 'utf8').replaceAll('UNFORMATTED', 'formatted'))",
  'if (e.FAKE_VANISH) rmSync(files[0])',
].join('\n')

let counter = 0
function fixture(files = { 'src/a.ts': 'export const a = UNFORMATTED\n' }) {
  const dir = join(base, `case-${++counter}`)
  const project = join(dir, 'project')
  const tmp = join(dir, 'tmp')
  mkdirSync(project, { recursive: true })
  mkdirSync(tmp)
  writeFileSync(join(dir, 'fake.mjs'), fakeSource)
  for (const [name, content] of Object.entries(files)) {
    const file = join(project, name)
    mkdirSync(dirname(file), { recursive: true })
    writeFileSync(file, content)
  }
  return { dir, project, tmp, fake: join(dir, 'fake.mjs') }
}

function run(c, args, env = {}, formatter = c.fake) {
  return spawnSync(process.execPath, [script, ...args], {
    cwd: c.project,
    encoding: 'utf8',
    timeout: 60000,
    env: {
      ...process.env,
      TEMP: c.tmp,
      TMP: c.tmp,
      PROTECTED_FORMAT_OXFMT: formatter,
      ...env,
    },
  })
}

const outDirs = (c) =>
  readdirSync(c.tmp)
    .filter((name) => name.startsWith('protected-format-'))
    .map((name) => join(c.tmp, name))
const readJson = (file) => JSON.parse(readFileSync(file, 'utf8'))
const read = (c, rel) => readFileSync(join(c.project, rel))

describe('arguments and the removed --write', () => {
  test('--write is rejected with exit 2 and names --prepare', () => {
    const c = fixture()
    const r = run(c, ['--write', '--all'])
    assert.equal(r.status, 2)
    assert.match(r.stderr, /--prepare/)
    assert.equal(outDirs(c).length, 0)
    assert.equal(
      read(c, 'src/a.ts').toString(),
      'export const a = UNFORMATTED\n'
    )
  })

  for (const [name, args] of [
    ['no arguments', []],
    ['mode without target', ['--check']],
    ['unknown flag', ['--check', '--force']],
    ['--all mixed with a file', ['--check', '--all', 'src/a.ts']],
    ['empty path', ['--check', '']],
  ]) {
    test(`bad usage exits 2: ${name}`, () => {
      const c = fixture()
      const r = run(c, args)
      assert.equal(r.status, 2)
      assert.equal(outDirs(c).length, 0)
    })
  }

  test('absolute paths are rejected even inside the root', () => {
    const c = fixture()
    const r = run(c, ['--check', join(c.project, 'src', 'a.ts')])
    assert.equal(r.status, 2)
    assert.match(r.stderr, /absolute/)
    assert.equal(outDirs(c).length, 0)
  })

  test('parent traversal is rejected', () => {
    const c = fixture()
    const r = run(c, ['--check', '../fake.mjs'])
    assert.equal(r.status, 2)
    assert.match(r.stderr, /outside project root/)
  })
})

describe('check and prepare', () => {
  test('--check reports differences with exit 1 and keeps no evidence', () => {
    const c = fixture()
    const r = run(c, ['--check', '--all'])
    assert.equal(r.status, 1)
    assert.match(r.stderr, /src.a.ts/)
    assert.equal(outDirs(c).length, 0)
    assert.equal(
      read(c, 'src/a.ts').toString(),
      'export const a = UNFORMATTED\n'
    )
  })

  test('--check exits 0 for already formatted files', () => {
    const c = fixture({ 'src/a.ts': 'export const a = formatted\n' })
    const r = run(c, ['--check', '--all'])
    assert.equal(r.status, 0)
    assert.equal(outDirs(c).length, 0)
  })

  test('--prepare writes exact candidates and never modifies the source', async () => {
    const { createHash } = await import('node:crypto')
    const c = fixture({
      'src/a.ts': 'export const a = UNFORMATTED\n',
      'src/b.ts': 'export const b = formatted\n',
    })
    const before = read(c, 'src/a.ts')
    const r = run(c, ['--prepare', '--all'])
    assert.equal(r.status, 0)
    assert.match(r.stdout, /not modified/)
    const dirs = outDirs(c)
    assert.equal(dirs.length, 1)
    const out = dirs[0]
    const result = readJson(join(out, 'result.json'))
    assert.equal(result.ok, true)
    assert.equal(result.sourceModified, false)
    assert.equal(result.total, 2)
    assert.equal(result.changed, 1)
    const entry = result.files.find((f) => f.file.endsWith('a.ts'))
    assert.equal(entry.changed, true)
    assert.equal(
      entry.originalSha256,
      createHash('sha256').update(before).digest('hex')
    )
    assert.equal(
      readFileSync(join(out, 'candidate', 'src', 'a.ts'), 'utf8'),
      'export const a = formatted\n'
    )
    assert.deepEqual(readFileSync(join(out, 'original', 'src', 'a.ts')), before)
    assert.match(readFileSync(join(out, 'changes.diff'), 'utf8'), /formatted/)
    assert.deepEqual(read(c, 'src/a.ts'), before)
    assert.equal(
      readFileSync(join(c.project, 'src', 'b.ts'), 'utf8'),
      'export const b = formatted\n'
    )
    assert.deepEqual(readdirSync(join(c.project, 'src')).sort(), [
      'a.ts',
      'b.ts',
    ])
  })

  test('--prepare with nothing to change reports zero changes', () => {
    const c = fixture({ 'src/a.ts': 'export const a = formatted\n' })
    const r = run(c, ['--prepare', '--all'])
    assert.equal(r.status, 0)
    const result = readJson(join(outDirs(c)[0], 'result.json'))
    assert.equal(result.changed, 0)
    assert.equal(result.ok, true)
  })

  test('a temp directory inside the project root is refused', () => {
    const c = fixture()
    const inside = join(c.project, 'tmp-inside')
    mkdirSync(inside)
    const r = run(c, ['--prepare', '--all'], { TEMP: inside, TMP: inside })
    assert.equal(r.status, 2)
    assert.deepEqual(readdirSync(inside), [])
    assert.equal(
      read(c, 'src/a.ts').toString(),
      'export const a = UNFORMATTED\n'
    )
  })
})

describe('header bytes are kept exactly', () => {
  const lines = [
    '/*',
    'Copyright (C) 2023-2026 QuantumNous',
    '',
    'GNU Affero General Public License text',
    '*/',
  ]
  for (const [name, bom, eol] of [
    ['LF', '', '\n'],
    ['CRLF', '', '\r\n'],
    ['BOM+LF', '﻿', '\n'],
    ['BOM+CRLF', '﻿', '\r\n'],
  ]) {
    test(`${name} header prefix is byte-identical in the candidate`, () => {
      const header = Buffer.from(bom + lines.join(eol) + eol + eol, 'utf8')
      const body = Buffer.from(`export const a = UNFORMATTED${eol}`)
      const original = Buffer.concat([header, body])
      const c = fixture({})
      writeFileSync(join(c.project, 'a.ts'), original)
      const r = run(c, ['--prepare', 'a.ts'])
      assert.equal(r.status, 0, r.stderr)
      const out = outDirs(c)[0]
      const candidate = readFileSync(join(out, 'candidate', 'a.ts'))
      assert.ok(candidate.subarray(0, header.length).equals(header))
      assert.equal(
        candidate.subarray(header.length).toString('utf8'),
        `export const a = formatted${eol}`
      )
      const entry = readJson(join(out, 'result.json')).files[0]
      assert.equal(entry.headerBytes, header.length)
      assert.ok(read(c, 'a.ts').equals(original))
    })
  }

  test('a file without a protected header is formatted as a whole', () => {
    const c = fixture({ 'a.ts': 'export const a = UNFORMATTED\n' })
    const r = run(c, ['--prepare', 'a.ts'])
    assert.equal(r.status, 0, r.stderr)
    const out = outDirs(c)[0]
    assert.equal(readJson(join(out, 'result.json')).files[0].headerBytes, 0)
    assert.equal(
      readFileSync(join(out, 'candidate', 'a.ts'), 'utf8'),
      'export const a = formatted\n'
    )
  })
})

describe('failures keep structured evidence and exit 3', () => {
  const unformatted = 'export const a = UNFORMATTED\n'

  function expectFailure(c, r, stage) {
    assert.equal(r.status, 3, r.stderr)
    const dirs = outDirs(c)
    assert.equal(dirs.length, 1)
    const result = readJson(join(dirs[0], 'result.json'))
    assert.equal(result.ok, false)
    assert.equal(result.stage, stage)
    assert.equal(typeof result.message, 'string')
    return { out: dirs[0], result }
  }

  test('formatter non-zero exit writes result.json and keeps the log', () => {
    const c = fixture()
    const r = run(c, ['--prepare', '--all'], { FAKE_FAIL: '1' })
    const { out, result } = expectFailure(c, r, 'format')
    assert.match(result.message, /exited with 3/)
    assert.match(
      readFileSync(join(out, 'formatter-0.log'), 'utf8'),
      /fake failure/
    )
    assert.equal(read(c, 'src/a.ts').toString(), unformatted)
  })

  test('--check also keeps evidence when the formatter fails', () => {
    const c = fixture()
    const r = run(c, ['--check', '--all'], { FAKE_FAIL: '1' })
    expectFailure(c, r, 'format')
    assert.equal(read(c, 'src/a.ts').toString(), unformatted)
  })

  test('formatter timeout fails within the budget and keeps evidence', () => {
    const c = fixture()
    const started = Date.now()
    const r = run(c, ['--prepare', '--all'], {
      FAKE_HANG: '1',
      PROTECTED_FORMAT_TIMEOUT_MS: '1500',
    })
    assert.ok(Date.now() - started < 30000)
    const { result } = expectFailure(c, r, 'format')
    assert.match(result.message, /timed out/)
    assert.equal(read(c, 'src/a.ts').toString(), unformatted)
  })

  test('a vanished formatter copy is a structured failure, not a crash', () => {
    const c = fixture()
    const r = run(c, ['--prepare', '--all'], { FAKE_VANISH: '1' })
    const { result } = expectFailure(c, r, 'collect-result')
    assert.match(result.message, /missing or unreadable/)
    assert.doesNotMatch(r.stderr, /at .*\.mjs:\d+/)
    assert.equal(read(c, 'src/a.ts').toString(), unformatted)
  })

  test('a source edited while formatting fails and the edit is preserved', () => {
    const c = fixture()
    const source = join(c.project, 'src', 'a.ts')
    const r = run(c, ['--prepare', '--all'], { FAKE_EDIT_SOURCE: source })
    const { out, result } = expectFailure(c, r, 'verify')
    assert.match(result.message, /changed during formatting/)
    assert.notEqual(result.originalSha256, result.currentSha256)
    assert.equal(readFileSync(source, 'utf8'), 'edited while formatting')
    assert.equal(readdirSync(out).includes('candidate'), false)
  })
})

describe('links, aliases and races never touch anything outside the candidate dir', () => {
  const unformatted = 'export const a = UNFORMATTED\n'

  function link(target, path) {
    symlinkSync(target, path, 'junction')
  }

  function outsideDir(c, content = 'outside UNFORMATTED\n') {
    const outside = join(c.dir, 'outside')
    mkdirSync(outside)
    writeFileSync(join(outside, 'a.ts'), content)
    return outside
  }

  test('--all rejects a link found while scanning', () => {
    const c = fixture()
    const outside = outsideDir(c)
    link(outside, join(c.project, 'linked'))
    const r = run(c, ['--prepare', '--all'])
    assert.equal(r.status, 2)
    assert.match(r.stderr, /link found/)
    assert.equal(outDirs(c).length, 0)
    assert.equal(
      readFileSync(join(outside, 'a.ts'), 'utf8'),
      'outside UNFORMATTED\n'
    )
  })

  test('an explicit path through a linked parent directory is rejected', () => {
    const c = fixture()
    const outside = outsideDir(c)
    link(outside, join(c.project, 'linked'))
    const r = run(c, ['--prepare', 'linked/a.ts'])
    assert.equal(r.status, 2)
    assert.match(r.stderr, /link in path/)
    assert.equal(outDirs(c).length, 0)
    assert.equal(
      readFileSync(join(outside, 'a.ts'), 'utf8'),
      'outside UNFORMATTED\n'
    )
  })

  test('a file symlink is rejected when the platform allows creating one', (t) => {
    const c = fixture()
    const outside = outsideDir(c)
    try {
      symlinkSync(join(outside, 'a.ts'), join(c.project, 'src', 'b.ts'), 'file')
    } catch (error) {
      t.skip(`cannot create file symlink: ${error.code}`)
      return
    }
    for (const args of [
      ['--prepare', '--all'],
      ['--prepare', 'src/b.ts'],
    ]) {
      const r = run(c, args)
      assert.equal(r.status, 2, r.stderr)
      assert.equal(outDirs(c).length, 0)
    }
    assert.equal(
      readFileSync(join(outside, 'a.ts'), 'utf8'),
      'outside UNFORMATTED\n'
    )
  })

  test('an in-root alias of node_modules cannot bypass the exclusion', () => {
    const c = fixture({ 'node_modules/pkg/index.js': 'var x = UNFORMATTED\n' })
    link(join(c.project, 'node_modules'), join(c.project, 'alias'))
    for (const args of [
      ['--prepare', '--all'],
      ['--prepare', 'alias/pkg/index.js'],
    ]) {
      const r = run(c, args)
      assert.equal(r.status, 2, r.stderr)
      assert.equal(outDirs(c).length, 0)
    }
    assert.equal(
      read(c, 'node_modules/pkg/index.js').toString(),
      'var x = UNFORMATTED\n'
    )
  })

  test('node_modules itself is skipped by --all and refused as an explicit path', () => {
    const c = fixture({
      'src/a.ts': 'export const a = UNFORMATTED\n',
      'node_modules/pkg/index.js': 'var x = UNFORMATTED\n',
    })
    const all = run(c, ['--check', '--all'])
    assert.equal(all.status, 1)
    assert.doesNotMatch(all.stderr, /node_modules/)
    const explicit = run(c, ['--check', 'node_modules/pkg/index.js'])
    assert.equal(explicit.status, 2)
    assert.match(explicit.stderr, /excluded/)
  })

  test('swapping a parent directory for a junction during formatting writes nothing outside', () => {
    const c = fixture()
    const outside = outsideDir(c)
    const swap = join(c.dir, 'swap.mjs')
    writeFileSync(
      swap,
      [
        "import { renameSync, symlinkSync } from 'node:fs'",
        'const src = process.env.SWAP_DIR',
        "renameSync(src, src + '.moved')",
        "symlinkSync(process.env.SWAP_TARGET, src, 'junction')",
      ].join('\n')
    )
    const r = run(
      c,
      ['--prepare', '--all'],
      { SWAP_DIR: join(c.project, 'src'), SWAP_TARGET: outside },
      swap
    )
    assert.equal(r.status, 3, r.stderr)
    const result = readJson(join(outDirs(c)[0], 'result.json'))
    assert.equal(result.ok, false)
    assert.equal(result.stage, 'verify')
    assert.equal(
      readFileSync(join(outside, 'a.ts'), 'utf8'),
      'outside UNFORMATTED\n'
    )
    assert.deepEqual(readdirSync(outside), ['a.ts'])
    assert.equal(
      readFileSync(join(c.project, 'src.moved', 'a.ts'), 'utf8'),
      unformatted
    )
  })

  test('a pre-existing file at the old temp name is never touched', () => {
    const c = fixture()
    const old = join(c.project, 'src', 'a.ts.protected-format.tmp')
    writeFileSync(old, 'keep me')
    const r = run(c, ['--prepare', 'src/a.ts'], { FAKE_FAIL: '1' })
    assert.equal(r.status, 3)
    assert.equal(readFileSync(old, 'utf8'), 'keep me')
    assert.deepEqual(readdirSync(join(c.project, 'src')).sort(), [
      'a.ts',
      'a.ts.protected-format.tmp',
    ])
  })

  test('no run leaves any file in the source tree except what was already there', () => {
    const c = fixture({
      'src/a.ts': unformatted,
      'src/b.css': 'a{color:UNFORMATTED}\n',
    })
    const list = () => readdirSync(c.project, { recursive: true }).sort()
    const before = list()
    for (const args of [
      ['--check', '--all'],
      ['--prepare', '--all'],
    ])
      run(c, args)
    run(c, ['--prepare', '--all'], { FAKE_FAIL: '1' })
    run(c, ['--prepare', '--all'], { FAKE_VANISH: '1' })
    assert.deepEqual(list(), before)
    assert.equal(read(c, 'src/a.ts').toString(), unformatted)
  })
})

// The real oxfmt binary only ever runs on the isolated copies the script makes
// inside the exclusive candidate directory, never on the fixture source tree.
const { existsSync } = await import('node:fs')
const realOxfmt = fileURLToPath(
  new URL(
    `../../node_modules/.bin/oxfmt${process.platform === 'win32' ? '.exe' : ''}`,
    import.meta.url
  )
)
const realConfig =
  JSON.stringify({ semi: false, singleQuote: true, endOfLine: 'lf' }) + '\n'

describe(
  'real oxfmt on isolated copies',
  { skip: !existsSync(realOxfmt) && 'oxfmt is not installed' },
  () => {
    const runReal = (c, args) => run(c, args, {}, realOxfmt)
    const realFixture = (files) =>
      fixture({ '.oxfmtrc.json': realConfig, ...files })

    test('check, prepare and a check of the candidate', () => {
      const original = 'export const a = {b:1};\n'
      const c = realFixture({ 'a.ts': original })
      assert.equal(runReal(c, ['--check', 'a.ts']).status, 1)
      assert.equal(outDirs(c).length, 0)
      const r = runReal(c, ['--prepare', 'a.ts'])
      assert.equal(r.status, 0, r.stderr)
      const candidate = readFileSync(
        join(outDirs(c)[0], 'candidate', 'a.ts'),
        'utf8'
      )
      assert.equal(candidate, 'export const a = { b: 1 }\n')
      assert.equal(read(c, 'a.ts').toString(), original)
      const next = realFixture({ 'a.ts': candidate })
      assert.equal(runReal(next, ['--check', 'a.ts']).status, 0)
    })

    for (const [name, bom, eol] of [
      ['LF', '', '\n'],
      ['CRLF', '', '\r\n'],
      ['BOM+LF', '﻿', '\n'],
      ['BOM+CRLF', '﻿', '\r\n'],
    ]) {
      test(`${name}: header bytes survive real oxfmt and the candidate passes a real check`, () => {
        const header = Buffer.from(
          `${bom}/*${eol}Copyright (C) 2023-2026 QuantumNous${eol}${eol}License text${eol}*/${eol}${eol}`,
          'utf8'
        )
        const original = Buffer.concat([
          header,
          Buffer.from(`export const a = {b:1};${eol}`),
        ])
        const c = realFixture({})
        writeFileSync(join(c.project, 'a.ts'), original)
        const r = runReal(c, ['--prepare', 'a.ts'])
        assert.equal(r.status, 0, r.stderr)
        const candidate = readFileSync(join(outDirs(c)[0], 'candidate', 'a.ts'))
        assert.ok(candidate.subarray(0, header.length).equals(header))
        assert.equal(
          candidate.subarray(header.length).toString('utf8'),
          'export const a = { b: 1 }\n'
        )
        assert.ok(read(c, 'a.ts').equals(original))
        const next = realFixture({})
        writeFileSync(join(next.project, 'a.ts'), candidate)
        assert.equal(runReal(next, ['--check', 'a.ts']).status, 0)
      })
    }
  }
)

// v3 (P2): .gitignore semantics. oxfmt 0.57.0 formats explicitly passed paths even
// when a .gitignore lists them, so the script itself must drop ignored targets.
// The 36 v2 tests above are the historical baseline and are intentionally untouched.
const { statSync } = await import('node:fs')
const posix = (name) => name.split(String.fromCharCode(92)).join('/')
const manifestFiles = (c) =>
  readJson(join(outDirs(c)[0], 'result.json'))
    .files.map((f) => posix(f.file))
    .sort()
const cssFiles = (dir) =>
  readdirSync(dir, { recursive: true })
    .map(posix)
    .filter((name) => name.endsWith('.css'))
    .sort()
const css = 'a{color:UNFORMATTED}\n'
const ignoreTree = {
  '.gitignore': 'gen/ignored/\nskip.css\n',
  'sub/.gitignore': 'local.css\n',
  'src/keep.css': css,
  'sub/keep.css': css,
  'sub/local.css': css,
  'gen/ignored/a.css': css,
  'skip.css': css,
}

describe('v3 ignore semantics (fake formatter)', () => {
  test('--prepare --all drops root-pattern, directory-pattern and nested-gitignore files', () => {
    const c = fixture(ignoreTree)
    const r = run(c, ['--prepare', '--all'])
    assert.equal(r.status, 0, r.stderr)
    assert.deepEqual(manifestFiles(c), ['src/keep.css', 'sub/keep.css'])
    assert.deepEqual(cssFiles(join(outDirs(c)[0], 'candidate')), [
      'src/keep.css',
      'sub/keep.css',
    ])
  })

  test('--check --all does not report an ignored file as a difference', () => {
    const c = fixture({
      '.gitignore': 'skip.css\n',
      'ok.css': 'a{color:formatted}\n',
      'skip.css': css,
    })
    const r = run(c, ['--check', '--all'])
    assert.equal(r.status, 0, r.stderr)
    assert.ok(!r.stderr.includes('skip.css'))
  })

  test('an explicitly named ignored file is skipped, not formatted', () => {
    const c = fixture(ignoreTree)
    const r = run(c, [
      '--prepare',
      'src/keep.css',
      'skip.css',
      'gen/ignored/a.css',
    ])
    assert.equal(r.status, 0, r.stderr)
    assert.deepEqual(manifestFiles(c), ['src/keep.css'])
  })

  test('ignored files keep their bytes and mtime', () => {
    const c = fixture(ignoreTree)
    const file = join(c.project, 'skip.css')
    const before = { bytes: readFileSync(file), mtime: statSync(file).mtimeMs }
    run(c, ['--prepare', '--all'])
    assert.ok(readFileSync(file).equals(before.bytes))
    assert.equal(statSync(file).mtimeMs, before.mtime)
  })
})

describe(
  'v3 ignore semantics (real oxfmt on isolated copies)',
  { skip: !existsSync(realOxfmt) && 'oxfmt is not installed' },
  () => {
    const json = '{"a":1,   "b":2}\n'
    const tree = {
      '.oxfmtrc.json': realConfig,
      '.gitignore': 'ignored.json\ngen/\n',
      'ok.json': json,
      'ignored.json': json,
      'gen/dir.json': json,
      'classic/src/c.ts': 'export const c = {b:1};\n',
    }

    test('ignored.json never enters manifest or candidate, normal JSON and Classic source still do', () => {
      const c = fixture(tree)
      const watched = ['ignored.json', 'gen/dir.json', 'ok.json']
      const before = watched.map((rel) => {
        const file = join(c.project, rel)
        return {
          file,
          bytes: readFileSync(file),
          mtime: statSync(file).mtimeMs,
        }
      })
      const r = run(c, ['--prepare', '--all'], {}, realOxfmt)
      assert.equal(r.status, 0, r.stderr)
      const files = manifestFiles(c)
      assert.ok(files.includes('ok.json'))
      assert.ok(files.includes('classic/src/c.ts'))
      assert.ok(!files.includes('ignored.json'))
      assert.ok(!files.includes('gen/dir.json'))
      const candidate = join(outDirs(c)[0], 'candidate')
      assert.ok(!existsSync(join(candidate, 'ignored.json')))
      assert.ok(!existsSync(join(candidate, 'gen', 'dir.json')))
      assert.equal(
        readFileSync(join(candidate, 'ok.json'), 'utf8'),
        '{ "a": 1, "b": 2 }\n'
      )
      for (const item of before) {
        assert.ok(readFileSync(item.file).equals(item.bytes))
        assert.equal(statSync(item.file).mtimeMs, item.mtime)
      }
    })

    test('--check reports ok.json but never mentions ignored files', () => {
      const c = fixture(tree)
      const r = run(c, ['--check', '--all'], {}, realOxfmt)
      assert.equal(r.status, 1)
      assert.ok(r.stderr.includes('ok.json'))
      assert.ok(!r.stderr.includes('ignored.json'))
      assert.ok(!r.stderr.includes('dir.json'))
    })

    test('--check with an already formatted file and an ignored unformatted file exits 0', () => {
      const c = fixture({
        '.oxfmtrc.json': realConfig,
        '.gitignore': 'ignored.json\n',
        'ok.json': '{ "a": 1, "b": 2 }\n',
        'ignored.json': json,
      })
      const r = run(c, ['--check', 'ok.json', 'ignored.json'], {}, realOxfmt)
      assert.equal(r.status, 0, r.stderr)
      assert.ok(!r.stderr.includes('ignored.json'))
    })
  }
)

// v3 (P3): failures during the initial scan and the first read must produce the
// documented exit 3 with a structured result.json and kept evidence. The failure
// is injected by a preload that lives in the fixture directory and only matches
// paths inside that fixture, so the real web tree is never involved.
const { pathToFileURL } = await import('node:url')
const injectSource = `import fs from 'node:fs'
import { syncBuiltinESMExports } from 'node:module'
import { resolve } from 'node:path'
const e = process.env
const hit = (p, key) => e[key] && resolve(String(p)) === resolve(e[key])
const fail = (message) => Object.assign(new Error('EACCES: ' + message), { code: 'EACCES' })
const readdir = fs.readdirSync
const readFile = fs.readFileSync
const realpath = fs.realpathSync
fs.readdirSync = (p, ...a) => {
  if (hit(p, 'INJECT_READDIR_FAIL')) throw fail('injected scan failure')
  return readdir(p, ...a)
}
fs.readFileSync = (p, ...a) => {
  if (hit(p, 'INJECT_READFILE_FAIL')) throw fail('injected read failure')
  return readFile(p, ...a)
}
fs.realpathSync = Object.assign((p, ...a) => {
  if (hit(p, 'INJECT_REALPATH_FAIL')) throw fail('injected realpath failure')
  return realpath(p, ...a)
}, realpath)
syncBuiltinESMExports()
`
function runInjected(c, args, env) {
  const inject = join(c.dir, 'inject.mjs')
  writeFileSync(inject, injectSource)
  return run(c, args, {
    NODE_OPTIONS: `--import ${pathToFileURL(inject).href}`,
    ...env,
  })
}

describe('v3 structured failures before formatting (fixture-only injection)', () => {
  const tree = {
    'src/a.ts': 'export const a = UNFORMATTED\n',
    'sub/b.ts': 'export const b = UNFORMATTED\n',
  }
  const expectStructured = (c, r, stage, file, text) => {
    assert.equal(r.status, 3, r.stderr)
    assert.ok(r.stderr.includes('Evidence kept at'), r.stderr)
    const dirs = outDirs(c)
    assert.equal(dirs.length, 1)
    const result = readJson(join(dirs[0], 'result.json'))
    assert.equal(result.ok, false)
    assert.equal(result.stage, stage)
    assert.equal(posix(result.file), file)
    assert.ok(result.message.includes(text), result.message)
    assert.ok(!existsSync(join(dirs[0], 'candidate')))
    assert.ok(!existsSync(join(dirs[0], 'work')))
  }

  test('directory scan failure exits 3 and keeps evidence without a candidate', () => {
    const c = fixture(tree)
    const r = runInjected(c, ['--prepare', '--all'], {
      INJECT_READDIR_FAIL: join(c.project, 'sub'),
    })
    expectStructured(c, r, 'scan', 'sub', 'injected scan failure')
  })

  test('first read failure exits 3 and does not continue with a partial file set', () => {
    const c = fixture(tree)
    const r = runInjected(c, ['--prepare', '--all'], {
      INJECT_READFILE_FAIL: join(c.project, 'src', 'a.ts'),
    })
    expectStructured(c, r, 'read', 'src/a.ts', 'injected read failure')
  })

  test('path validation realpath failure exits 3 with structured evidence', () => {
    const c = fixture(tree)
    const r = runInjected(c, ['--check', 'src/a.ts'], {
      INJECT_REALPATH_FAIL: join(c.project, 'src', 'a.ts'),
    })
    expectStructured(c, r, 'scan', 'src/a.ts', 'injected realpath failure')
  })

  test('failed runs leave source bytes and mtime unchanged', () => {
    const c = fixture(tree)
    const file = join(c.project, 'src', 'a.ts')
    const before = { bytes: readFileSync(file), mtime: statSync(file).mtimeMs }
    runInjected(c, ['--prepare', '--all'], {
      INJECT_READFILE_FAIL: file,
    })
    assert.ok(readFileSync(file).equals(before.bytes))
    assert.equal(statSync(file).mtimeMs, before.mtime)
  })
})

// v4 (P3): the first lstat in the explicit-path parent chain must not turn
// EACCES/EPERM into "not found" (exit 2). The injection lives in the fixture dir
// and only matches one exact fixture path.
const injectLstatSource = `import fs from 'node:fs'
import { syncBuiltinESMExports } from 'node:module'
import { resolve } from 'node:path'
const lstat = fs.lstatSync
fs.lstatSync = Object.assign((p, ...a) => {
  const target = process.env.INJECT_LSTAT_FAIL
  if (target && resolve(String(p)) === resolve(target)) {
    throw Object.assign(new Error('EACCES: injected lstat failure'), { code: 'EACCES' })
  }
  return lstat(p, ...a)
}, lstat)
syncBuiltinESMExports()
`

describe('v4 first lstat failure on explicit paths (fixture-only injection)', () => {
  const tree = { 'src/a.ts': 'export const a = UNFORMATTED\n' }
  const runLstat = (c, args, env) => {
    const inject = join(c.dir, 'inject-lstat.mjs')
    writeFileSync(inject, injectLstatSource)
    return run(c, args, {
      NODE_OPTIONS: `--import ${pathToFileURL(inject).href}`,
      ...env,
    })
  }

  test('EACCES on an existing path exits 3 with structured evidence', () => {
    const c = fixture(tree)
    const r = runLstat(c, ['--check', 'src/a.ts'], {
      INJECT_LSTAT_FAIL: join(c.project, 'src', 'a.ts'),
    })
    assert.equal(r.status, 3, r.stderr)
    assert.ok(r.stderr.includes('Evidence kept at'), r.stderr)
    const dirs = outDirs(c)
    assert.equal(dirs.length, 1)
    const result = readJson(join(dirs[0], 'result.json'))
    assert.equal(result.ok, false)
    assert.equal(result.stage, 'scan')
    assert.equal(posix(result.file), 'src/a.ts')
    assert.ok(result.message.includes('injected lstat failure'), result.message)
  })

  test('a truly missing path still exits 2 without evidence', () => {
    const c = fixture(tree)
    const r = runLstat(c, ['--check', 'src/missing.ts'], {
      INJECT_LSTAT_FAIL: join(c.project, 'src', 'a.ts'),
    })
    assert.equal(r.status, 2, r.stderr)
    assert.ok(r.stderr.includes('not found'), r.stderr)
    assert.equal(outDirs(c).length, 0)
  })

  test('failed run leaves source bytes and mtime unchanged', () => {
    const c = fixture(tree)
    const file = join(c.project, 'src', 'a.ts')
    const before = { bytes: readFileSync(file), mtime: statSync(file).mtimeMs }
    runLstat(c, ['--check', 'src/a.ts'], { INJECT_LSTAT_FAIL: file })
    assert.ok(readFileSync(file).equals(before.bytes))
    assert.equal(statSync(file).mtimeMs, before.mtime)
  })
})

// v4 (P2): when the project root is a subdirectory of an existing Git work tree,
// the parent repository's ignore rules must still apply. The fixture creates its
// own throwaway repository around the fixture project; the real repository is
// never involved. Without a Git context the v3 isolated fallback stays in force.
function parentRepo(files, parentFiles) {
  const c = fixture(files)
  for (const [name, content] of Object.entries(parentFiles)) {
    writeFileSync(join(c.dir, name), content)
  }
  const git = spawnSync('git', ['init', '-q'], {
    cwd: c.dir,
    encoding: 'utf8',
    timeout: 60000,
    env: { ...process.env, TEMP: c.tmp, TMP: c.tmp },
  })
  assert.equal(git.status, 0, git.stderr)
  return c
}

describe('v4 parent repository ignore context (fake formatter)', () => {
  const parentRules = '.cache/\nskills-lock.json\n*.gen.css\n!keep.gen.css\n'
  const tree = {
    'src/keep.css': css,
    'src/a.gen.css': css,
    'src/keep.gen.css': css,
    '.cache/profile.json': '{"a":UNFORMATTED}\n',
    'skills-lock.json': '{"a":UNFORMATTED}\n',
    'sub/.gitignore': 'local.css\n',
    'sub/keep.css': css,
    'sub/local.css': css,
  }

  test('--prepare --all applies parent root, pattern, nested and negated rules', () => {
    const c = parentRepo(tree, { '.gitignore': parentRules })
    const r = run(c, ['--prepare', '--all'])
    assert.equal(r.status, 0, r.stderr)
    assert.deepEqual(manifestFiles(c), [
      'src/keep.css',
      'src/keep.gen.css',
      'sub/keep.css',
    ])
    const candidate = join(outDirs(c)[0], 'candidate')
    assert.ok(!existsSync(join(candidate, '.cache')))
    assert.ok(!existsSync(join(candidate, 'skills-lock.json')))
  })

  test('explicitly named files ignored by the parent repository are skipped', () => {
    const c = parentRepo(tree, { '.gitignore': parentRules })
    const r = run(c, [
      '--prepare',
      'src/keep.css',
      'skills-lock.json',
      '.cache/profile.json',
      'src/a.gen.css',
    ])
    assert.equal(r.status, 0, r.stderr)
    assert.deepEqual(manifestFiles(c), ['src/keep.css'])
  })

  test('--check --all exits 0 when only parent-ignored files are unformatted', () => {
    const c = parentRepo(
      {
        'ok.css': 'a{color:formatted}\n',
        '.cache/profile.json': '{"a":UNFORMATTED}\n',
        'skills-lock.json': '{"a":UNFORMATTED}\n',
      },
      { '.gitignore': '.cache/\nskills-lock.json\n' }
    )
    const r = run(c, ['--check', '--all'])
    assert.equal(r.status, 0, r.stderr)
    assert.ok(!r.stderr.includes('skills-lock.json'))
    assert.ok(!r.stderr.includes('profile.json'))
  })

  test('a tracked but ignored file is still skipped and the repository is not modified', () => {
    const c = parentRepo(tree, { '.gitignore': parentRules })
    const add = spawnSync('git', ['add', '-f', 'project/skills-lock.json'], {
      cwd: c.dir,
      encoding: 'utf8',
      timeout: 60000,
      env: { ...process.env, TEMP: c.tmp, TMP: c.tmp },
    })
    assert.equal(add.status, 0, add.stderr)
    const gitFile = (name) => join(c.dir, '.git', name)
    const snapshot = () => ({
      index: readFileSync(gitFile('index')),
      config: readFileSync(gitFile('config')),
    })
    const lock = join(c.project, 'skills-lock.json')
    const before = {
      git: snapshot(),
      bytes: readFileSync(lock),
      mtime: statSync(lock).mtimeMs,
    }
    const r = run(c, ['--prepare', '--all'])
    assert.equal(r.status, 0, r.stderr)
    assert.ok(!manifestFiles(c).includes('skills-lock.json'))
    const after = snapshot()
    assert.ok(after.index.equals(before.git.index))
    assert.ok(after.config.equals(before.git.config))
    assert.ok(readFileSync(lock).equals(before.bytes))
    assert.equal(statSync(lock).mtimeMs, before.mtime)
  })
})

describe(
  'v4 parent repository ignore context (real oxfmt on isolated copies)',
  { skip: !existsSync(realOxfmt) && 'oxfmt is not installed' },
  () => {
    const json = '{"a":1,   "b":2}\n'
    const parentRules = '.cache/\nskills-lock.json\n'
    // --all also scans the fixture's own .oxfmtrc.json, so these cases use the config
    // text exactly as oxfmt formats it. Shadows the compact outer realConfig; no assertion changes.
    const realConfig =
      '{ "semi": false, "singleQuote": true, "endOfLine": "lf" }\n'

    test('--check --all exits 0 with parent-ignored unformatted files, like native oxfmt', () => {
      const c = parentRepo(
        {
          '.oxfmtrc.json': realConfig,
          'ok.json': '{ "a": 1, "b": 2 }\n',
          '.cache/profile.json': json,
          'skills-lock.json': json,
        },
        { '.gitignore': parentRules }
      )
      const r = run(c, ['--check', '--all'], {}, realOxfmt)
      assert.equal(r.status, 0, r.stderr)
      assert.ok(!r.stderr.includes('skills-lock.json'))
      assert.ok(!r.stderr.includes('profile.json'))
    })

    test('--prepare --all keeps parent-ignored files out of manifest and candidate', () => {
      const c = parentRepo(
        {
          '.oxfmtrc.json': realConfig,
          'ok.json': json,
          '.cache/profile.json': json,
          'skills-lock.json': json,
        },
        { '.gitignore': parentRules }
      )
      const watched = ['.cache/profile.json', 'skills-lock.json', 'ok.json']
      const before = watched.map((rel) => {
        const file = join(c.project, rel)
        return {
          file,
          bytes: readFileSync(file),
          mtime: statSync(file).mtimeMs,
        }
      })
      const r = run(c, ['--prepare', '--all'], {}, realOxfmt)
      assert.equal(r.status, 0, r.stderr)
      // --all scans the fixture's own formatted .oxfmtrc.json as a normal JSON file; only
      // the parent-ignored .cache/profile.json and skills-lock.json must be absent.
      assert.deepEqual(manifestFiles(c), ['.oxfmtrc.json', 'ok.json'])
      const candidate = join(outDirs(c)[0], 'candidate')
      assert.ok(!existsSync(join(candidate, '.cache')))
      assert.ok(!existsSync(join(candidate, 'skills-lock.json')))
      assert.equal(
        readFileSync(join(candidate, 'ok.json'), 'utf8'),
        '{ "a": 1, "b": 2 }\n'
      )
      for (const item of before) {
        assert.ok(readFileSync(item.file).equals(item.bytes))
        assert.equal(statSync(item.file).mtimeMs, item.mtime)
      }
    })
  }
)
