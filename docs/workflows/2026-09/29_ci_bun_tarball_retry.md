# CI 中 Bun 截断下载重试修复

## 背景与证据边界

PR #287 的前端 CI 在两个不同 GitHub 托管 Runner 和区域中，均于 Classic 的
`bun install --frozen-lockfile` 阶段失败，错误为：

```text
error: Fail extracting tarball for "mermaid"
error: Fail extracting tarball from mermaid
```

根前端依赖安装、类型检查和 772 项测试均已通过，Classic 构建尚未开始。日志没有
`ENOSPC`、磁盘或 I/O 错误、HTTP 状态、registry 超时及网络 errno。官方
`mermaid@11.6.0` tarball 可完整解压，SHA512 与 `web/classic/bun.lock` 一致。

上述生产 CI 日志本身没有保存响应体或底层 socket 状态，因此不能单凭日志断言某次
下载具体在哪个字节断开。不过，Bun 1.3.14 存在已确认缺陷：tarball 响应在收到完整
HTTP 头后、达到 `Content-Length` 前中断时，会把下载失败误报为解包失败且不执行既有
重试。Bun 上游 PR `oven-sh/bun#40063` 修复了该路径，并进入 Bun 1.4.1。

## 受控复现

使用实际 `mermaid@11.6.0` tarball（13,911,361 字节、733 个文件）和同一 SHA512，
本地临时 registry 对第一次请求返回完整 `Content-Length`，仅发送前 1 MiB 后正常关闭
连接，第二次请求返回完整 tarball。两次测试均使用同一个 Bun v1 锁文件和
`--frozen-lockfile`：

- Bun 1.3.14：精确输出上述两行 `Fail extracting tarball`，退出码为 1，未重试。
- Bun 1.4.1：记录 `ConnectionClosed`，执行内建 `Retrying 1/5`，第二次下载成功，
  退出码为 0，安装结果为 `mermaid@11.6.0`。

该夹具证实工具链修复覆盖已知失败机制；它不是对两次 GitHub CI 底层连接状态的追溯。

## 修改范围

- `.github/workflows/ci.yml`：将 CI Bun 固定版本从 1.3.14 更新为 1.4.1。
- 不修改 `package.json`、`bun.lock` 或任何业务依赖版本。
- 保留 `bun install --frozen-lockfile`、类型检查、测试和 Classic 生产构建，不增加跳过
  检查、无限重试或依赖缓存清理。
- 不改生产配置，不触发部署。

## 验收

1. 受控截断下载夹具中，Bun 1.3.14 失败、Bun 1.4.1 经一次有限重试成功。
2. 使用仓库现有根前端和 Classic 锁文件执行 `bun install --frozen-lockfile`，锁文件不应
   发生变化。
3. GitHub Actions 前端作业应完成根前端安装、类型检查、测试、Classic 安装和构建。
4. `git diff --check` 通过，提交范围仅包含 CI 工作流和本记录。

## 已知限制与回滚

- CI 通过只能证明当前 Runner 上完整链路成功，不能证明网络以后永不抖动；Bun 1.4.1
  的有限重试用于吸收已识别的中途断开。
- 若新工具链产生独立兼容问题，可仅回滚工作流中的 Bun 固定版本；依赖锁文件无需回滚。
