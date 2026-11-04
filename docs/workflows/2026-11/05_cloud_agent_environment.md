# Cloud Agent 本地开发环境

## 目标与范围

为 NewAPI-Mao 记录可复用的 Linux Cloud Agent 本地启动方法。范围是工具链、依赖安装和开发进程：Go API 使用 SQLite，Default 前端使用 Rsbuild。Classic 仅安装依赖并满足 Go 嵌入要求，不启动其 Vite 服务；不改任一模板的业务代码、业务接口或生产部署。

## 方案

- Go 使用 `go.mod` 中的 1.25.1，Bun 使用 CI 中的 1.4.1。镜像预装版本属于平台状态，执行前需重新核对。
- 原 Cloud Agent 的 `install`、`start` 是平台环境配置，不是仓库中的脚本，未随本 PR 提交。它们分别安装依赖、生成缺失的嵌入页，以及按端口监听情况用 tmux 启动 API 和 Default。是否自动执行取决于平台配置是否保存生效。
- 不依赖外部环境配置时，可按以下步骤手动启动。每个命令块均从仓库根目录执行；本次平台路径为 `/agent/repos/newapi-mao`，其他检出使用实际路径。

### 依赖与嵌入页

先安装上述版本的 Go 和 Bun。若 Bun 安装在 `~/.bun/bin`，在每个新 shell 中显式补充当前进程的 `PATH`，避免依赖 `~/.bashrc`：

```bash
export PATH="$HOME/.bun/bin:$PATH"
go version
bun --version
go mod download
(cd relaykit && GOWORK=off go mod download)
(cd web && bun install --frozen-lockfile)
(cd web/classic && bun install --frozen-lockfile)

for dist in web/dist web/classic/dist; do
  if [ ! -s "$dist/index.html" ]; then
    mkdir -p "$dist"
    printf '%s\n' '<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>开发占位页</title><body>请访问 Default 开发服务器 :5173</body></html>' > "$dist/index.html"
  fi
done
```

占位页仅满足 `main.go` 的 `go:embed` 编译要求，不提供正式前端。重复执行保留已有非空首页；发布前需真实构建两个模板。

### 手动启动

在第一个终端启动 API：

```bash
env SQL_DSN= LOG_SQL_DSN= REDIS_CONN_STRING= SESSION_COOKIE_TRUSTED_URL= \
  SQLITE_PATH=one-api.db SESSION_COOKIE_SECURE=false PORT=3000 go run .
```

这些进程级空值用于覆盖已有环境变量和 `.env`。仅设置 `SESSION_COOKIE_SECURE=false` 而保留 `SESSION_COOKIE_TRUSTED_URL` 会被 `common/session_cookie.go` 拒绝；未覆盖 `SQL_DSN` 或 `LOG_SQL_DSN` 则可能连接外部数据库。

在第二个终端启动 Default：

```bash
export PATH="$HOME/.bun/bin:$PATH"
cd web
VITE_REACT_APP_SERVER_URL=http://127.0.0.1:3000 bun run dev -- --host 0.0.0.0 --port 5173
```

访问 `http://127.0.0.1:5173/`，API 状态为 `http://127.0.0.1:3000/api/status`。Rsbuild 代理 `/api`、`/mj`、`/pg`；端口冲突时可能选择其他端口，以终端输出为准。重复启动前应先确认原进程和服务身份，不能仅凭端口已占用就认定启动成功。手动进程用各自终端的 Ctrl+C 停止。

## 安全与兼容性

默认数据库是仓库根目录的 `one-api.db`，已被 gitignore，该文件只属于本机开发数据。上述 HTTP Cookie 设置仅适用于隔离开发环境；HTTPS 使用需按 `common/session_cookie.go` 同时配置安全 Cookie 与可信 HTTPS 来源。环境配置不写入密钥。Docker Compose 里的 PostgreSQL 与 Redis 仍可供需要它们的开发使用，但不是 Cloud Agent 的默认依赖。

回退时停止开发进程即可；本次文档变更没有数据迁移，不自动删除本地数据库或已有构建产物。

## 验证计划

依赖准备连续执行两次，确认保留已有构建首页。启动 API 与 Default 后检查状态接口，通过初始化向导创建根用户，并用密码登录进入后台。业务代码变更时可执行根模块 `go build ./...`、`go test -timeout 60s ./common/`，独立模块 `cd relaykit && GOWORK=off go build ./... && GOWORK=off go test -timeout 60s ./...`，以及 `web/` 下的 `bun run typecheck`、`bun run test`。

## 验证结果

以下为原 PR 在 Cloud Agent 上记录的验证结果，手动命令是本次审查补充的等价启动步骤，不代表在其他平台重新执行过初始化验收：

- 安装脚本连续两次均成功，第二次没有重新下载依赖。
- 启动后 `GET /api/status` 与 `http://127.0.0.1:5173/` 返回 200。再次执行启动脚本时服务已在监听，脚本直接返回。
- 浏览器完成初始化向导：数据库检测为 SQLite，创建根用户后出现 “System initialized successfully”，随后进入 `/dashboard/overview`。`POST /api/user/login` 对已创建用户返回成功会话，角色为根用户。
- 根模块 `go build ./...` 成功。`relaykit` 在 `GOWORK=off` 下构建和测试通过。`go test ./common/` 通过。Default 前端 `bun run typecheck` 通过，Vitest 175 个文件、1105 项测试通过。
- 草稿构建 `bld-20261104-d95aa275-387a-45e7-bb39-900d399b1b91` 的 install 退出码为 0。两台从该草稿构建启动的机器都有 Go 1.25.1 与 Bun 1.4.1，但 `/tmp/cursor/start-user/` 不存在；当时未观察到自动启动，不能据此保证其他镜像或已保存配置的行为。原环境中手动执行同一启动脚本时，`:3000` 与 `:5173` 返回 200。

## PR 审查修正

- 核对 `go.mod`、CI、`main.go`、Rsbuild、Makefile 与 Compose：工具版本、双模板嵌入、代理路径和可选数据库说明与源码一致。
- 修正 Cookie 可信来源未清空、数据库与 `.env` 可能覆盖 SQLite 默认值的问题；补充不依赖平台私有脚本的命令及本地开发边界。
- 新增工作记录已登记到开发文档索引。审查仅修改文档，验证 Markdown 格式、新增相对链接、Bash 语法与 Git 空白差异；原 PR 前后端 CI 均通过，不重复运行业务测试。
