# GitHub 大仓库技能导入超时

## 问题

zzapi 导入 `zhaoxuya520/reverse-skill` 时提示：

```text
读取技能文件 "skills/dotnet-reverse/SKILL.md" 失败：无法读取 GitHub 技能仓库，请稍后重试
```

## 根因

- 该仓库在技能根目录内有 506 个可导入文本文件。旧实现逐个串行下载原始文件，整个导入共用
  30 秒期限；本地对真实 GitHub 复现，正好在 30 秒时失败。
- 期限到达后，正在进行的请求返回 `context deadline exceeded`，被统一包装为“请稍后重试”，
  失败路径只是期限到达时恰好在下载的文件，与该文件本身无关。
- 并发修复后又暴露第二个问题：仓库中有一份 Markdown 示例包含真实 NUL 字节（空字节截断
  payload 示例）。旧校验把 NUL 与非 UTF-8 合并拒绝，导致整个仓库无法导入。

## 修改

- `service/skill_repository.go`
  - 先按树声明大小校验 64 MiB / 4096 文件上限，再以 8 路并发下载原始文件；任一文件失败立即
    取消其余下载，并按路径顺序报告首个真实错误。
  - 导入总期限改为 90 秒，低于 Cloudflare 100 秒源站响应上限。
  - 期限到达时返回“超过 90 秒时限（已读取 N / M 个文本文件）”，并建议改用 `tree` 目录链接；
    客户端取消直接返回取消，不再误报网络错误。
  - 连接失败、响应体中断和 GitHub 502/503/504 各最多重试两次（间隔 0.5 秒、1 秒）；404、
    403、超限等其他状态立即失败。
- `service/skill_import.go`：ZIP 与仓库导入共用编码校验。非 UTF-8 仍拒绝，并提示先转换编码；
  NUL 字节转写为字面 `\x00`，PostgreSQL TEXT 可保存且不改变示例含义。
- 前端无改动：Default 与 Classic 的导入请求均未设置客户端超时。

## 兼容性

- 导入结果、草稿结构、来源记录和保存接口不变；只改变下载方式、期限、错误文案和 NUL 处理。
- 原先因 NUL 被拒绝的技能包现在可导入，正文中对应位置显示为 `\x00`。
- 一键更新复用同一导入路径，同样受益。

## 验证

- 新增 `service/skill_repository_concurrency_test.go`：
  - 并发下载不超过 8 路且确实并行；
  - 期限到达报告时限与进度，而不是“请稍后重试”；
  - ZIP 与仓库导入均转写 NUL、拒绝非 UTF-8；
  - 502 重试后成功、404 不重试。
- `TestRepositoryImportEnforcesAggregateDownloadBudget` 改为断言在下载任何原始文件前按声明大小
  拒绝超限包。
- 本地对真实 `zhaoxuya520/reverse-skill` 连续导入并保存两次：约 16 秒完成下载，89 个技能，
  草稿 5.6 MiB，来源记录完整；所有技能均有用途说明，上架校验通过。
- `go vet` 与 `model`、`service`、`controller`、`router` 测试通过。
- 未运行 `-race`：本机无 C 编译器，cgo 不可用。各下载协程只写自己的切片下标，主协程在
  `WaitGroup` 结束后读取；测试夹具的请求记录已加锁。
