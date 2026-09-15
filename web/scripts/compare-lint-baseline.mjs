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
import { execFileSync, spawnSync } from 'node:child_process'
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// 在本工作树的忽略目录内比较快照；不切换分支、不修改源码、不安装依赖。
const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = path.dirname(webRoot)
const [baseRef = 'origin/custom-main', headRef = 'HEAD', bun = 'bun'] =
  process.argv.slice(2)
const git = (...args) =>
  execFileSync('git', args, { cwd: repoRoot, encoding: 'utf8' }).trim()
const base = git('rev-parse', '--verify', `${baseRef}^{commit}`)
const head = git('rev-parse', '--verify', `${headRef}^{commit}`)
// 不放入 node_modules：依赖目录会改变 oxlint 对源码循环导入的解析行为。
const cache = path.join(repoRoot, '.local-tests')
mkdirSync(cache, { recursive: true })
const output = mkdtempSync(path.join(cache, 'lint-baseline-'))
const changed = new Set(
  git('diff', '--name-only', base, head, '--', 'web').split('\n')
)
const runs = {}

for (const [name, commit] of [
  ['base', base],
  ['head', head],
]) {
  const archive = path.join(output, `${name}.tar`)
  const snapshot = path.join(output, name)
  mkdirSync(snapshot)
  git('archive', '--format=tar', `--output=${archive}`, commit, 'web')
  execFileSync('tar', ['-xf', archive, '-C', snapshot])
  const cwd = path.join(snapshot, 'web')
  symlinkSync(
    path.join(webRoot, 'node_modules'),
    path.join(cwd, 'node_modules'),
    process.platform === 'win32' ? 'junction' : 'dir'
  )
  runs[name] = {}
  for (const [scope, extra] of [
    ['all', []],
    ['default', ['--ignore-pattern', 'classic']],
  ]) {
    const args = ['run', 'lint', ...extra, '--format', 'json']
    const result = spawnSync(bun, args, {
      cwd,
      encoding: 'utf8',
      timeout: 60000,
      maxBuffer: 32 * 1024 * 1024,
    })
    if (result.error) throw result.error
    if (result.status !== 0 && result.status !== 1) {
      throw new Error(`lint exit ${result.status}`)
    }
    writeFileSync(path.join(output, `${name}-${scope}.json`), result.stdout)
    writeFileSync(
      path.join(output, `${name}-${scope}.stderr.txt`),
      result.stderr
    )
    const report = JSON.parse(result.stdout)
    if (!Array.isArray(report.diagnostics)) {
      throw new Error('缺少 diagnostics，不能把执行失败当作无错误')
    }
    runs[name][scope] = {
      exit: result.status,
      errors: report.diagnostics.filter((d) => d.severity === 'error'),
    }
  }
}

// 行号会随合法增删移动，以文件、规则、文案的多重集合比较，保留重复诊断数量。
const signature = (d) =>
  JSON.stringify([d.filename.replaceAll('\\', '/'), d.code, d.message])
const summaries = {}
for (const scope of ['all', 'default']) {
  const remaining = [...runs.base[scope].errors]
  const added = []
  for (const error of runs.head[scope].errors) {
    const index = remaining.findIndex(
      (candidate) => signature(candidate) === signature(error)
    )
    if (index < 0) added.push(error)
    else remaining.splice(index, 1)
  }
  const changedFileErrors = runs.head[scope].errors.filter((d) =>
    changed.has(`web/${d.filename.replaceAll('\\', '/')}`)
  )
  summaries[scope] = {
    baseExit: runs.base[scope].exit,
    headExit: runs.head[scope].exit,
    baseErrors: runs.base[scope].errors.length,
    headErrors: runs.head[scope].errors.length,
    addedErrors: added.length,
    removedErrors: remaining.length,
    changedFileErrors: changedFileErrors.length,
  }
  writeFileSync(
    path.join(output, `${scope}-difference.json`),
    JSON.stringify({ added, removed: remaining, changedFileErrors }, null, 2)
  )
}
const summary = {
  base,
  head,
  bun: execFileSync(bun, ['--version'], { encoding: 'utf8' }).trim(),
  oxlint: JSON.parse(
    readFileSync(path.join(webRoot, 'node_modules/oxlint/package.json'), 'utf8')
  ).version,
  scopes: summaries,
}
writeFileSync(
  path.join(output, 'summary.json'),
  JSON.stringify(summary, null, 2)
)
console.log(JSON.stringify({ ...summary, output }, null, 2))
// 历史错误仍使完整 lint 失败；本脚本退出码仅代表是否新增或遗留变更文件错误。
if (Object.values(summaries).some((s) => s.addedErrors || s.changedFileErrors)) {
  process.exitCode = 1
}
