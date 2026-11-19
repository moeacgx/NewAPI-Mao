# 格式化脚本受保护头安全（v2 修复计划）

状态：实现与测试已由另一名 Codex 独立审查（v4 复核 56/56，P2/P3 已关闭）；安全契约仅候选输出、`--check` 只读、`--write` 拒绝，本轮不执行该脚本任何模式；不部署 zzapi。本文由「修复计划」更新为「计划 + 实现结果 + 验证结果」，结果见下文「实现结果与用法」「验证结果」「最终 SHA256」。

## 背景

早先一次格式化事故导致 783 个文件丢失版权头。第一版修复脚本 `web/scripts/format-with-protected-headers.mjs`（SHA256 8E13E90E159682A55BDF199264909CD256F9352D89D6D7BA00C233E48275203E）被两名独立审查拒绝，共 5 项 P2 和若干 P3。本文记录 v2 的修复计划与最终契约。第一版「哈希校验后 wx 写临时文件再 rename」的设计已被废弃。

## 冻结基线

| 文件                                                           | SHA256                                                           |
| -------------------------------------------------------------- | ---------------------------------------------------------------- |
| web/scripts/format-with-protected-headers.mjs                  | 8E13E90E159682A55BDF199264909CD256F9352D89D6D7BA00C233E48275203E |
| `web/scripts/__tests__/format-with-protected-headers.test.mjs` | 200483C5A87B80508EEB864F0E2B738BFD4F1B9B1CD832C6495BE3C4F3121F3F |
| web/package.json                                               | 9D6FC96865D89A30BA1CBE496AE76C14147C25AAFBCD3AFF2DC3AB4FD42D3022 |
| 本文（修改前）                                                 | B563E3AEEB5642AB4A14338EB461B2A4DCA1210DE9675718661433BC0BC4A2CE |
| 审查报告 formatter-header-review-f3-final/report.md            | BA195F6B325CE3B207399FBA3DA4EF1E5332ADEE5DB69B6B63F3C440B3229DAE |
| 审查报告 formatter-header-review-724-final/review.md           | D3D447C0A42FB96C1CF5C4D8CA673F61B770BA68F9F182669D5B640FFDBBE1CC |

四个待改文件的基线备份在 `D:/Temp/formatter-header-repair-v2/backup/`。

## 必须处理的缺陷

P2

1. 最终 SHA 校验与 rename 之间的并发编辑被静默覆盖，脚本仍退出 0。
2. 格式化期间父目录被换成 junction，写入落到根目录之外，退出 0。
3. 根内 junction 别名（如 src/link 指向 node_modules/private）绕过 node_modules 排除，退出 0。
4. `wx` 创建失败后，catch 无条件 `rmSync` 删除了并非本次创建的既有同名临时文件。
5. BOM+CRLF 版权头未被 headerPattern 识别，真实 oxfmt 把头部 CRLF 改成 LF。

P3

- 绝对路径：文档称拒绝，实际根内绝对路径被接受，需统一为拒绝。
- 格式化器非零退出、副本消失（ENOENT 未捕获崩溃）都没有结构化结果和证据保留。
- 旧测试断言的临时文件名 `${target}.protected-format.tmp` 与实际名称（含 pid）不一致。
- 内部格式化超时 120s，需改为 60s 以内。
- 真实 oxfmt（endOfLine=lf）会把正文换成 LF；头部必须逐字节保留，正文允许遵循格式化器配置。

## 安全契约（协调者选定）

Node 在本平台没有可靠的条件替换能力，不用更多哈希或 realpath 检查伪装原子 CAS。因此：

- 脚本永不写回共享源文件，也不通过源文件父目录写任何临时文件。
- 新增 `--prepare`：在独占创建的临时目录输出候选文件、原文件摘要、result.json 清单和可审差异。
- `--check` 保持只读，有差异退出 1。
- 旧 `--write` 直接以非零退出并提示改用 `--prepare`。这是明确的接口改动，不是行为修复。
- 原子 CAS 未实现，也不提供承诺安全的自动 apply 步骤；应用候选需人工审阅后手动完成。
- 路径收集拒绝绝对路径和含链接的父链，并检查真实路径是否落入排除目录；读取期间文件变化则检测并失败退出。不宣称能抵御任意恶意程序。
- 受信的真实 oxfmt 只运行在隔离副本上。
- 头部逐字节保留（含 BOM、LF/CRLF）；格式化器失败、超时、副本消失均写结构化结果并保留证据，非零退出。
- package.json：`format` 改为安全候选命令，`format:check` 仍为只读检查。

## 测试计划

所有测试只在独占目录 `D:/Temp/formatter-header-repair-v2` 内运行，每个进程不超过 60s，TEMP/TMP 指向 D 盘。

1. 红证据：用备份的旧脚本，在最小隔离夹具中复现 5 项 P2，并用报告里的真实原输入与边界。
2. 新契约：LF / CRLF / BOM+LF / BOM+CRLF 头部精确前缀；候选内容准确；源文件、根外文件、既有同名 tmp 全程不变。
3. 格式化器失败、副本消失、超时的结构化证据；静态与动态链接、node_modules 别名；候选目录独占创建；并发编辑保存；参数边界（含旧 `--write` 被拒）。
4. 用已安装的真实 oxfmt 0.57.0 做 check / prepare / 候选 check，不只用 fake formatter。
5. 不在真实 web 目录运行脚本，不做全量 format/build/typecheck。

## 历史记录与证据来源限制

- 旧的 8 个测试是假绿：并发测试检查了脚本从不创建的临时文件名，junction 测试只覆盖预先存在的情形。
- 两名审查员共用 `D:/Temp/formatter-header-review`，无法证明没有同名覆盖，其证据只作线索，本次修复以自己独占目录中的复现为准。

## 实现结果与用法

- `--check`：只读，有差异退出 1，不留证据目录。
- `--prepare`：在系统临时目录下独占创建 `protected-format-*`，写入 `original/`、`candidate/`、`changes.diff`、`result.json`，源文件不被修改，成功退出 0。
- `--write`：退出 2 并提示改用 `--prepare`。这是接口改动，不是缺陷修复。
- 参数只接受 `--all` 或相对路径。绝对路径、`..`、含链接的父链、排除目录（含指向其内的别名）一律退出 2。
- 退出码：0 成功；1 `--check` 有差异；2 用法或路径被拒；3 运行失败（格式化器非零或超时、副本消失、读取或格式化期间源文件变化）。退出 3 时保留候选目录，`result.json` 含 `ok:false` 与 `stage`。
- 真实 oxfmt 只运行在候选目录内的 `work/` 副本上；内部超时上限 50s，可用 `PROTECTED_FORMAT_TIMEOUT_MS` 缩短。
- 变化检测范围：读取前后比较 lstat，格式化后比较源文件摘要。窗口之外的变化无法发现。
- 使用：在 `web/` 运行 `bun run format`，阅读输出目录中的 `changes.diff` 与 `result.json`。如需采纳，由人工确认源文件摘要等于 `originalSha256` 后手工复制候选文件。该步骤不是原子 CAS；原子 CAS 未实现，脚本不提供也不承诺安全的自动 apply。

## 验证结果

全部在独占目录 `D:/Temp/formatter-header-repair-v2` 内执行，TEMP/TMP 指向该目录，每个进程不超过 60s。

- 红证据：`red-repro.mjs` 对备份的旧脚本在隔离夹具中复现 8/8，结果见 `red/red-results.json`：并发编辑被覆盖（退出 0）；junction 替换写到根外（退出 0）；junction 别名绕过 node_modules 排除（退出 0）；既有 tmp 被删除（退出 1）；BOM+CRLF 头被真实 oxfmt 改写（退出 0）；根内绝对路径被接受（退出 1，非 2）；副本消失 ENOENT 崩溃且无 result.json（退出 1）；格式化器非零退出无 result.json（退出 3）。
- 新脚本：`node --test web/scripts/__tests__/format-with-protected-headers.test.mjs`。第一次 36 项通过 35、失败 1：node_modules 用例的夹具只有 node_modules 文件，`--all` 正确报「no files to format」退出 2，属测试夹具错误，补入 `src/a.ts` 后脚本未改。第二次 36/36 通过，退出 0，约 9.1s（`test-run-2.log`）。其中 5 项使用真实 oxfmt 0.57.0：check、prepare、候选 check，以及 LF / CRLF / BOM+LF / BOM+CRLF 头前缀逐字节一致。
- 负对照：同一套测试对备份旧脚本运行（排除 timeout 用例，避免旧脚本 120s 超时），31 项通过 10、失败 21，退出 1（`neg-control-old-script.log`）。局限：旧脚本不识别 `--prepare`，21 项失败主要由接口改变造成；通过的 3 项（文件符号链接、node_modules 别名、无遗留文件）因旧 CLI 在参数解析处即退出而没有证明力。缺陷的红证据以上面的 `red-results.json` 为准。

## 最终 SHA256（冻结时）

| 文件                                                           | SHA256                                                           |
| -------------------------------------------------------------- | ---------------------------------------------------------------- |
| web/scripts/format-with-protected-headers.mjs                  | 471890BB954A23C9DCFB112074D1B8795032B83EF416B056EF663BF443E3CAF4 |
| `web/scripts/__tests__/format-with-protected-headers.test.mjs` | D5946DE907E9E9394FCA5E61D8007AB651E9101B8E28A552B9D0665165A0D70D |
| web/package.json                                               | 84CA60A9AF7DD2B4AA1271C4A84BD3F56A851C29ECAD1C200C8B500BD97712C9 |

本文自身的 SHA256 因自引用不写入表中，在交付消息中给出。相对冻结基线，`web/package.json` 只改了 `format`（`--write --all` 改为 `--prepare --all`），`format:check` 未变。

## v3 修复（针对 v2 独立审查的 P2/P3）

上文 v2 的 36/36、红证据与负对照作为历史保留，不改写。v3 新增回归单独计数，不与 v2 基线混算。候选只输出、`--check` 零写、`--write` 退出 2 的契约均未改变。

- P2：`--all` 遗漏 `.gitignore`。真实 oxfmt 0.57.0 对显式传入的路径仍会格式化被忽略文件，只复制 `.gitignore` 不够。脚本在读取与复制之前用 git 自带匹配器过滤目标：`git check-ignore --no-index --stdin -z`，工作目录为项目根，`GIT_DIR` 指向候选目录内的空 bare 仓库，全局与系统配置指向候选目录内的空文件。根目录与嵌套 `.gitignore`、目录规则对 `--all` 和显式路径都生效。被忽略文件不进入 manifest、candidate 与 work，`--check` 也不会报告其差异。`.oxfmtrc.json` 的排除仍由真实 oxfmt 在 `work/` 内处理。匹配器失败按 `stage=scan` 退出 3，不会带着不完整的文件集合继续。
- 实现中发现：Windows 上 `os.devNull` 为 `\\.\nul`，git 无法将其作为配置路径打开（`fatal: unable to access`），所有运行因此退出 3。已改用空文件。
- P3：初始目录扫描、路径校验的 `lstat`/`realpath` 与首次 `readFile` 失败原先抛出裸异常。现在写出 `result.json`（`ok:false`、`stage`、`file`、`message`），保留证据目录并退出 3。扫描与路径校验失败的 `stage` 为 `scan`，读取失败为 `read`。
- 测试注入：通过 `NODE_OPTIONS --import` 加载写在夹具目录内的预加载脚本，只对环境变量指定的夹具路径覆盖 `readdirSync`、`readFileSync`、`realpathSync`，不触及真实 `web/`。
- 测试数字（均在独占 D 盘目录，TEMP/TMP 指向其中，每进程不超过 60s）：先红，v3 共 11 项，通过 2、失败 9，退出 1，约 2.4s；修复后 v3 11/11，退出 0，约 10.1s；全文件 47/47，退出 0，约 22.1s，即 v2 的 36 项加 v3 的 11 项。其中 3 项使用真实 oxfmt（`ignored.json` 与 `gen/dir.json` 不进入 manifest 与 candidate，`ok.json` 与 Classic 源文件仍被处理，`--check` 只提 `ok.json`）。
- 过程记录：转绿过程中一度出现 7 项失败，原因是上面的 `devNull` 问题，并非断言被放宽。
- 格式检查：只用已安装的底层 oxfmt，不经过本脚本。脚本与测试在 `web/` 下检查，退出 0。文档在带有 `web/.oxfmtrc.json` 的 D 盘副本上检查，退出 0。oxfmt 曾把表格中的 `__tests__` 改成 `**tests**`，现已用反引号包住以避免。格式化后重跑全文件测试，47/47，退出 0，约 19.8s。
- v3 冻结时 SHA256：`web/scripts/format-with-protected-headers.mjs` 为 50B7BD9074EEF20B821643244F1753D7D69F9F160E3CC5EF383D95A2C51187AA，`web/scripts/__tests__/format-with-protected-headers.test.mjs` 为 7CEED06F4B00556A1DFBCAB5DA1711399E7AA3CFC5CB5DC6301EF494D9FA4278，`web/package.json` 未改动，仍为 84CA60A9AF7DD2B4AA1271C4A84BD3F56A851C29ECAD1C200C8B500BD97712C9。本文自身的 SHA256 在交付消息中给出。

## v4 修复（针对 v3 独立审查的 P2/P3）

v2 的 36 项基线与 v3 的 11 项回归作为历史保留，不改写；v4 回归单独计数。候选只输出、`--check` 零写、`--write` 退出 2 的契约均未改变。

- P3：路径参数的首次 `lstat` 失败原先一律当作“路径不存在”以退出 2 拒绝。现在只有 `ENOENT` 与 `ENOTDIR` 按“not found”退出 2；其他错误（例如 `EACCES`）写出 `result.json`（`ok:false`、`stage=scan`、`file`、`message`），保留证据目录并退出 3，不会带着不完整的文件集合继续。
- P2：v3 的 `dropIgnored` 在候选目录内建立空 bare 仓库并以 `GIT_DIR` 指向它，使 `web/` 成为工作树根，父仓库的 `.gitignore` 链被截断。v4 先只读执行 `git rev-parse --is-inside-work-tree`（工作目录为项目根，不设置 `GIT_DIR` 与 `GIT_WORK_TREE`）。位于工作树内时，直接用 `git check-ignore --no-index --stdin -z` 在父仓库上下文中判定，父级根规则、嵌套规则、否定规则、`info/exclude` 与 `core.excludesFile` 都生效，且不写索引与配置；不在工作树内时沿用 v3 的空 bare 仓库回退。探测失败（“not a git repository”以外的错误）按 `stage=scan` 退出 3。已跟踪但被忽略的文件同样被跳过。
- 测试夹具只在 D 盘独占目录内自建父仓库（`git init` 作用于夹具的父目录），不触及真实仓库；断言父仓库 `.git/index` 与 `.git/config` 前后字节一致，源文件字节与 mtime 不变。
- 测试数字（TEMP/TMP 指向证据目录内的独占子目录，每进程不超过 60s）：P3 先红 3 项（通过 2、失败 1），修复后 7/7（含 v3 结构化失败用例），退出 0，约 0.7s；P2 先红 6 项（通过 0、失败 6），退出 1，约 2.8s；修复后 v4 父仓库用例 6/6，退出 0，约 2.2s；全文件 56/56，12 个 suite，退出 0，约 18.7s，即 v2 的 36 项、v3 的 11 项、v4 的 3 项 lstat 与 6 项父仓库用例。
- 过程记录：真实 oxfmt 的两项用例转绿过程中，夹具写入的紧凑 `.oxfmtrc.json` 被 `--all` 当作普通 JSON 扫描并报告未格式化。这是夹具问题，不是脚本缺口。处理方式是改用 oxfmt 格式化后的配置文本，并把 manifest 预期改为严格等于 `.oxfmtrc.json` 与 `ok.json`；对被父仓库忽略的 `.cache/profile.json` 与 `skills-lock.json` 的排除断言（不进入 manifest 与 candidate）未放宽。
- 格式检查：只用已安装的底层 oxfmt，不经过本脚本。脚本与测试先在 D 盘副本上格式化（仅换行调整），确认真实文件与副本起点一致后写回，在 `web/` 下检查退出 0。格式化后重跑全文件测试，56/56，12 个 suite，退出 0，约 19s。文档在带有 `web/.oxfmtrc.json` 的 D 盘副本上检查，路径中的 `__tests__` 一律用反引号包住。
- v4 冻结时 SHA256：`web/scripts/format-with-protected-headers.mjs` 为 C0DA3E2C7A59EBEDC01255A1EEE91E11934D78B1DBF58EF00BA1BA6AB7EEC183，`web/scripts/__tests__/format-with-protected-headers.test.mjs` 为 9888D6FF99B83DEC8E921139B01692B83E1D8D701C248B5BEDEB46776C91FFE0，`web/package.json` 未改动，仍为 84CA60A9AF7DD2B4AA1271C4A84BD3F56A851C29ECAD1C200C8B500BD97712C9。本文自身的 SHA256 在交付消息中给出。

## 双模板（Default / Classic）开发工具影响

- Default（`web/`）：`bun run format` 只产出候选目录，不再改写源码；`format:check` 仍为只读检查。`--all` 从 `web/` 递归，排除目录列表不含 `classic`，因此会遍历 `web/classic/src`。
- Classic（`web/classic/package.json`）：检索 `format`、`oxfmt`、`protected` 无匹配，没有自己的格式化脚本，不受 package 脚本改动影响。
- 其他文档仅在历史记录中提到本脚本，无需改动。

## 未验证边界

- 未在真实 `web/` 目录运行脚本（任务禁止），因此真实目录的文件数量、耗时、`web/classic` 是否适用同一份 `.oxfmtrc.json` 均未验证。
- 未在 Linux/macOS 验证；junction 用例只在 Windows 上验证；网络盘及 case-insensitive 之外的文件系统行为未验证。
- 不宣称能抵御任意恶意程序；变化检测只覆盖读取前后与格式化后的摘要比较。
- 候选应用流程为人工步骤，未提供自动 apply，原子 CAS 未实现。

## 范围外

依赖锁、业务源码、locale、共享 docs/developer/README.md（由协调者整合）、提交、推送、部署。
