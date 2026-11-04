# Cloud Agent 本地开发环境

## 目标与范围

为 NewAPI-Mao 准备可重复启动的 Cloud Agent 环境。范围是本机工具链、依赖安装和默认开发进程：Go API 使用 SQLite，Default 前端使用 Rsbuild。不改业务接口，不启动 Classic 的 Vite，也不把 PostgreSQL 或 Redis 放进默认启动。

## 方案

- Go 使用 `go.mod` 中的 1.25.1。Bun 固定为 CI 中的 1.4.1，并链接到 `/usr/local/bin/bun`，因为登录 shell 不会读取 `~/.bashrc`。
- `install` 在 `web/` 与 `web/classic/` 执行 `bun install --frozen-lockfile`，下载根模块和 `relaykit`（`GOWORK=off`）依赖，并在缺少构建产物时写入 `web/dist/index.html` 与 `web/classic/dist/index.html`。这两个目录被 gitignore，但 `go:embed` 编译时必须存在。
- `start` 在端口未监听时用 tmux 启动 `go run .`（`:3000`）和 `cd web && bun run dev -- --host 0.0.0.0 --port 5173`。本地 HTTP 设置 `SESSION_COOKIE_SECURE=false`。重复执行时如果端口已响应则跳过。

## 安全与兼容性

默认数据库是仓库根目录的 `one-api.db`，已被 gitignore。该文件只属于本机开发数据。环境配置不写入密钥。Docker Compose 里的 PostgreSQL 与 Redis 仍可供需要它们的开发使用，但不是 Cloud Agent 的默认依赖。

## 验证计划

安装脚本连续执行两次。启动脚本拉起 API 与 Default 前端后，再执行一次以确认幂等。通过初始化向导创建根用户，并用密码登录进入后台。同时运行 `go build ./...`、`relaykit` 的 `GOWORK=off go build ./...` 与 `go test ./...`、`web` 的 `bun run typecheck` 与 `bun run test`，以及 `go test ./common/`。

## 验证结果

- 安装脚本连续两次均成功，第二次没有重新下载依赖。
- 启动后 `GET /api/status` 与 `http://127.0.0.1:5173/` 返回 200。再次执行启动脚本时服务已在监听，脚本直接返回。
- 浏览器完成初始化向导：数据库检测为 SQLite，创建根用户后出现 “System initialized successfully”，随后进入 `/dashboard/overview`。`POST /api/user/login` 对已创建用户返回成功会话，角色为根用户。
- 根模块 `go build ./...` 成功。`relaykit` 在 `GOWORK=off` 下构建和测试通过。`go test ./common/` 通过。Default 前端 `bun run typecheck` 通过，Vitest 175 个文件、1105 项测试通过。
