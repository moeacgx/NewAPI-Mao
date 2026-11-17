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
import { spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import {
  lstatSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  realpathSync,
  rmSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import {
  dirname,
  extname,
  isAbsolute,
  join,
  relative,
  resolve,
  sep,
} from 'node:path'

// Contract: this script NEVER writes to the source tree. It formats isolated
// copies and either reports differences (--check) or leaves reviewable
// candidates in a freshly created temporary directory (--prepare). Atomic
// compare-and-swap replacement of shared files is not implemented, so applying
// a candidate is a manual, reviewed step.
const usage = [
  'Usage: node scripts/format-with-protected-headers.mjs --check|--prepare (--all | <relative-file>...)',
  '  --check    read-only; exit 1 when formatting would change a file',
  '  --prepare  write candidates, digests and a diff to a new temp directory; source files stay untouched',
  '  Exit codes: 0 ok, 1 check found differences, 2 bad usage or rejected path, 3 run failed (evidence kept)',
].join('\n')

const [mode, ...args] = process.argv.slice(2)

if (mode === '--write') {
  console.error(
    '--write was removed: this script no longer writes source files.\n' +
      'Use --prepare to get reviewable candidates in a temporary directory, then apply them manually after review.\n' +
      usage
  )
  process.exit(2)
}
if (
  (mode !== '--check' && mode !== '--prepare') ||
  args.length === 0 ||
  (args.includes('--all') && args.length > 1) ||
  args.some(
    (arg) =>
      arg === '' ||
      arg.includes('\0') ||
      (arg.startsWith('--') && arg !== '--all')
  )
) {
  console.error(usage)
  process.exit(2)
}

const root = process.cwd()
const realRoot = realpathSync(root)
const excludedDirs = new Set([
  '.git',
  '.tanstack',
  'build',
  'coverage',
  'dist',
  'node_modules',
])
const headerExtensions = new Set([
  '.cjs',
  '.cts',
  '.js',
  '.jsx',
  '.mjs',
  '.mts',
  '.ts',
  '.tsx',
])
const headerPattern =
  /^\/\*\r?\nCopyright \(C\)[\s\S]*?QuantumNous[\s\S]*?\*\/(?:\r?\n)+/
const bomBytes = Buffer.from([0xef, 0xbb, 0xbf])
const MAX_BYTES = 10 * 1024 * 1024
// Each child process must finish within 60s, so the formatter gets at most 50s.
const FORMATTER_TIMEOUT_MS = Math.min(
  Number(process.env.PROTECTED_FORMAT_TIMEOUT_MS) || 50000,
  50000
)

const sha = (buffer) => createHash('sha256').update(buffer).digest('hex')
const isExcluded = (rel) =>
  rel.split(/[\\/]/).some((part) => excludedDirs.has(part.toLowerCase()))
const escapesRoot = (rel) =>
  rel === '..' || rel.startsWith('..' + sep) || isAbsolute(rel)

// Everything this script writes goes under `out`, created exclusively by
// mkdtemp. It must live outside the project so a source walk never sees it.
const realTmp = realpathSync(tmpdir())
if (!escapesRoot(relative(realRoot, realTmp))) {
  console.error('Rejected path: TEMP/TMP must be outside the project root')
  process.exit(2)
}
const out = mkdtempSync(join(realTmp, 'protected-format-'))
const cleanup = () => rmSync(out, { recursive: true, force: true })

function reject(message) {
  cleanup()
  console.error(`Rejected path: ${message}`)
  process.exit(2)
}

function writeResult(data) {
  writeFileSync(
    join(out, 'result.json'),
    JSON.stringify({ mode, ...data }, null, 2)
  )
}

function fail(stage, message, extra = {}) {
  writeResult({ ok: false, stage, message, ...extra })
  console.error(`Protected format failed at ${stage}: ${message}`)
  console.error(`Evidence kept at ${out}`)
  process.exit(3)
}

// The protected header is everything up to and including the blank lines after
// the copyright comment, plus a leading BOM. Its bytes are never given to the
// formatter and are re-attached verbatim, so BOM and CRLF survive exactly.
function splitHeader(file, original) {
  const none = { prefix: Buffer.alloc(0), body: original }
  if (!headerExtensions.has(extname(file).toLowerCase())) return none
  const bom = original.subarray(0, 3).equals(bomBytes) ? 3 : 0
  const match = headerPattern.exec(original.subarray(bom).toString('latin1'))
  if (!match) return none
  const length = bom + match[0].length
  return {
    prefix: original.subarray(0, length),
    body: original.subarray(length),
  }
}

// Collect: only relative paths are accepted. Links anywhere in the parent chain
// and excluded directories (checked again on the real path) are rejected.
const formatExtensions = new Set([
  ...headerExtensions,
  '.css',
  '.json',
  '.jsonc',
  '.md',
])
const targets = []

function walk(dir) {
  let entries
  try {
    entries = readdirSync(dir, { withFileTypes: true })
  } catch (error) {
    const where = relative(root, dir).split(sep).join('/') || '.'
    fail('scan', `cannot read directory ${where}: ${error.message}`, {
      file: where,
    })
  }
  for (const entry of entries) {
    const abs = join(dir, entry.name)
    const rel = relative(root, abs)
    if (entry.isSymbolicLink()) reject(`link found while scanning: ${rel}`)
    if (entry.isDirectory()) {
      if (!excludedDirs.has(entry.name.toLowerCase())) walk(abs)
    } else if (
      entry.isFile() &&
      formatExtensions.has(extname(entry.name).toLowerCase())
    ) {
      targets.push(rel)
    }
  }
}

if (args[0] === '--all') {
  walk(root)
} else {
  for (const arg of args) {
    if (isAbsolute(arg) || /^[A-Za-z]:/.test(arg)) {
      reject(`absolute paths are not accepted: ${arg}`)
    }
    const abs = resolve(root, arg)
    const rel = relative(root, abs)
    if (rel === '' || escapesRoot(rel)) reject(`outside project root: ${arg}`)
    if (isExcluded(rel)) reject(`excluded directory: ${arg}`)
    let current = root
    for (const part of rel.split(sep)) {
      current = join(current, part)
      let stat
      try {
        stat = lstatSync(current)
      } catch (error) {
        if (error.code === 'ENOENT' || error.code === 'ENOTDIR') {
          reject(`not found: ${arg}`)
        }
        fail('scan', `cannot inspect ${arg}: ${error.message}`, {
          file: rel.split(sep).join('/'),
        })
      }
      if (stat.isSymbolicLink()) reject(`link in path: ${arg}`)
    }
    let realAbs
    let leaf
    try {
      realAbs = realpathSync(abs)
      leaf = lstatSync(abs)
    } catch (error) {
      fail('scan', `cannot resolve ${arg}: ${error.message}`, {
        file: rel.split(sep).join('/'),
      })
    }
    const realRel = relative(realRoot, realAbs)
    if (escapesRoot(realRel) || isExcluded(realRel)) {
      reject(`resolves outside the allowed tree: ${arg}`)
    }
    if (!leaf.isFile() || !formatExtensions.has(extname(abs).toLowerCase())) {
      reject(`not a supported file: ${arg}`)
    }
    targets.push(rel)
  }
}

// Ignore semantics: oxfmt formats explicitly passed paths even when a
// .gitignore lists them, so ignored targets are dropped here, before anything is
// read or copied. git's own matcher is used (root and nested .gitignore, with
// --no-index so tracked state is irrelevant) against an empty bare repository
// under `out`; global and system git config are disabled for determinism. A
// probe failure is a structured failure, never a silent partial file set.
function dropIgnored(list) {
  const gitDir = join(out, 'gitignore-probe')
  const emptyConfig = join(out, 'empty-gitconfig')
  writeFileSync(emptyConfig, '')
  const quiet = {
    ...process.env,
    GIT_CONFIG_GLOBAL: emptyConfig,
    GIT_CONFIG_SYSTEM: emptyConfig,
    GIT_CONFIG_NOSYSTEM: '1',
    XDG_CONFIG_HOME: join(out, 'xdg'),
  }
  delete quiet.GIT_DIR
  delete quiet.GIT_WORK_TREE
  // v4: when root already lives inside a Git work tree (for example web/ in the
  // project repository), keep that repository's read-only ignore context so parent
  // .gitignore files, info/exclude and core.excludesFile still apply. Only without
  // a Git context is the isolated empty bare repository used.
  const inside = spawnSync('git', ['rev-parse', '--is-inside-work-tree'], {
    cwd: root,
    encoding: 'utf8',
    env: quiet,
    timeout: 20000,
  })
  if (
    inside.error ||
    (inside.status !== 0 &&
      !String(inside.stderr).includes('not a git repository'))
  ) {
    fail(
      'scan',
      `cannot detect git context: ${inside.error?.message ?? tail(inside.stderr)}`
    )
  }
  const inWorkTree = inside.status === 0 && inside.stdout.trim() === 'true'
  if (!inWorkTree) {
    const init = spawnSync('git', ['init', '--bare', '-q', gitDir], {
      encoding: 'utf8',
      env: quiet,
      timeout: 20000,
    })
    if (init.error || init.status !== 0) {
      fail(
        'scan',
        `cannot prepare .gitignore matcher: ${init.error?.message ?? tail(init.stderr)}`
      )
    }
  }
  const probe = spawnSync(
    'git',
    ['check-ignore', '--no-index', '--stdin', '-z'],
    {
      cwd: root,
      encoding: 'utf8',
      env: inWorkTree
        ? quiet
        : { ...quiet, GIT_DIR: gitDir, GIT_WORK_TREE: root },
      input: list.map((file) => file.split(sep).join('/')).join('\0') + '\0',
      timeout: 30000,
    }
  )
  if (probe.error || (probe.status !== 0 && probe.status !== 1)) {
    fail(
      'scan',
      `cannot evaluate .gitignore: ${probe.error?.message ?? tail(probe.stderr)}`
    )
  }
  const ignored = new Set(probe.stdout.split('\0').filter(Boolean))
  return list.filter((file) => !ignored.has(file.split(sep).join('/')))
}

const tail = (text) => String(text ?? '').slice(-2000)

// Read each file once and compare lstat before and after the read. A change
// during the read fails the run with evidence; this detects ordinary concurrent
// edits only and is not a defence against a hostile local process.
const items = []
for (const file of dropIgnored([...new Set(targets)].sort())) {
  const abs = join(root, file)
  let before
  let original
  let after
  try {
    before = lstatSync(abs)
    if (!before.isFile() || before.size > MAX_BYTES) {
      reject(`not a regular file within ${MAX_BYTES} bytes: ${file}`)
    }
    original = readFileSync(abs)
    after = lstatSync(abs)
  } catch (error) {
    fail('read', `cannot read ${file}: ${error.message}`, {
      file: file.split(sep).join('/'),
    })
  }
  if (
    original.length !== before.size ||
    after.size !== before.size ||
    after.mtimeMs !== before.mtimeMs
  ) {
    fail('read', `file changed while being read: ${file}`, { file })
  }
  items.push({ file, original, digest: sha(original) })
}
if (items.length === 0) reject('no files to format')

// Format: the formatter only ever sees copies under out/work, never the project.
const work = join(out, 'work')
mkdirSync(work)
try {
  writeFileSync(
    join(work, '.oxfmtrc.json'),
    readFileSync(join(root, '.oxfmtrc.json'))
  )
} catch (error) {
  if (error.code !== 'ENOENT') {
    fail('setup', `cannot copy .oxfmtrc.json: ${error.message}`)
  }
}
for (const item of items) {
  const { prefix, body } = splitHeader(item.file, item.original)
  item.prefix = prefix
  item.copy = join(work, item.file)
  mkdirSync(dirname(item.copy), { recursive: true })
  writeFileSync(item.copy, body)
}

const binary =
  process.env.PROTECTED_FORMAT_OXFMT ||
  join(
    root,
    'node_modules',
    '.bin',
    process.platform === 'win32' ? 'oxfmt.exe' : 'oxfmt'
  )
const viaNode = ['.mjs', '.js', '.cjs'].includes(extname(binary).toLowerCase())
const command = viaNode ? process.execPath : binary
const commandArgs = viaNode ? [binary] : []
const deadline = Date.now() + FORMATTER_TIMEOUT_MS

for (let i = 0; i < items.length; i += 100) {
  const chunk = items.slice(i, i + 100).map((item) => item.file)
  const remaining = deadline - Date.now()
  if (remaining <= 0) {
    fail('format', 'formatter time budget exhausted', {
      timeoutMs: FORMATTER_TIMEOUT_MS,
    })
  }
  const run = spawnSync(command, [...commandArgs, '--write', ...chunk], {
    cwd: work,
    encoding: 'utf8',
    timeout: remaining,
    windowsHide: true,
  })
  writeFileSync(
    join(out, `formatter-${i / 100}.log`),
    `status=${run.status} signal=${run.signal}\n${run.stdout ?? ''}\n${run.stderr ?? ''}`
  )
  const evidence = {
    files: chunk.length,
    status: run.status,
    signal: run.signal,
    stdoutTail: tail(run.stdout),
    stderrTail: tail(run.stderr),
  }
  if (run.error?.code === 'ETIMEDOUT') {
    fail(
      'format',
      `formatter timed out after ${FORMATTER_TIMEOUT_MS}ms`,
      evidence
    )
  }
  if (run.error) fail('format', run.error.message, evidence)
  if (run.status !== 0) {
    fail('format', `formatter exited with ${run.status}`, evidence)
  }
}

// Rebuild each candidate as original header bytes + formatted body, and make
// sure the source did not change while the formatter ran.
const results = []
for (const item of items) {
  let formatted
  try {
    formatted = readFileSync(item.copy)
  } catch (error) {
    fail(
      'collect-result',
      `formatter copy missing or unreadable: ${item.file} (${error.code})`,
      { file: item.file }
    )
  }
  let now
  try {
    now = readFileSync(join(root, item.file))
  } catch (error) {
    fail(
      'verify',
      `source unreadable after formatting: ${item.file} (${error.code})`,
      {
        file: item.file,
      }
    )
  }
  if (sha(now) !== item.digest) {
    fail('verify', `source changed during formatting: ${item.file}`, {
      file: item.file,
      originalSha256: item.digest,
      currentSha256: sha(now),
    })
  }
  const candidate = Buffer.concat([item.prefix, formatted])
  results.push({
    file: item.file,
    changed: !candidate.equals(item.original),
    headerBytes: item.prefix.length,
    originalSha256: item.digest,
    candidateSha256: sha(candidate),
    candidate,
  })
}

// Finish: --check reports and cleans up; --prepare keeps candidates for review.
const changed = results.filter((result) => result.changed)
const summary = results.map(({ candidate, ...rest }) => rest)

if (mode === '--check') {
  cleanup()
  if (changed.length > 0) {
    console.error(`Formatting would change ${changed.length} file(s):`)
    for (const result of changed) console.error(`  ${result.file}`)
    process.exit(1)
  }
  console.log(`OK: ${results.length} file(s) already formatted`)
  process.exit(0)
}

const originals = new Map(items.map((item) => [item.file, item.original]))
for (const result of changed) {
  for (const [folder, content] of [
    ['original', originals.get(result.file)],
    ['candidate', result.candidate],
  ]) {
    const target = join(out, folder, result.file)
    mkdirSync(dirname(target), { recursive: true })
    writeFileSync(target, content)
  }
}

let diff = 'skipped: nothing changed'
if (changed.length > 0) {
  const run = spawnSync(
    'git',
    ['diff', '--no-index', '--no-color', '--', 'original', 'candidate'],
    {
      cwd: out,
      encoding: 'utf8',
      timeout: 20000,
      windowsHide: true,
      maxBuffer: 64 * 1024 * 1024,
    }
  )
  if (run.error) {
    diff = `unavailable: ${run.error.message}`
  } else {
    writeFileSync(join(out, 'changes.diff'), run.stdout)
    diff = 'changes.diff'
  }
}

writeResult({
  ok: true,
  outDir: out,
  sourceModified: false,
  total: results.length,
  changed: changed.length,
  diff,
  note: 'Source files were not modified. Review original/ and candidate/ (or changes.diff) and apply by hand; atomic compare-and-swap is not implemented.',
  files: summary,
})
console.log(
  `Candidates for ${changed.length} of ${results.length} file(s) written to ${out}`
)
console.log(
  'Source files were not modified. Review changes.diff and apply manually.'
)
process.exit(0)
