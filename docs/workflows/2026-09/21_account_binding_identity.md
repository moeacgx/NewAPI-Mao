# 账号绑定身份与邮箱更新检查

## 报告与核验边界

用户报告已登录账号填写邮箱和验证码绑定后出现新账号。两套前端该操作均调用
`POST /api/oauth/email/bind`，后端从 UserAuth 读取用户 ID，并无创建用户或签发新登录态分支。
本次三应用可用日志样本内未找到邮箱绑定请求，且没有账号 ID/发生时间，不能把其他路径的
缺陷描述成已还原这次现场。用真实路由与数据库回归验证绑定前后用户 ID、数量、资金及会话身份。

## 确认的问题与方案

- 邮箱持久化原先用完整旧 User 快照调用通用 UpdateWithTx，会回写与邮箱无关的角色、状态、分组和社交身份。
  修复为按现存正整数 ID 锁定并仅更新 email，保留邮箱唯一性串行保护，缺失/删除用户拒绝，回读最新用户再更新缓存。
- 邮箱验证码原先只校验不消费，可在有效期内重复绑定。绑定专用调用改为在同一互斥锁内校验并消费，错误验证码不消耗；匹配后即消费，数据库或缓存失败需要重新获取验证码。
- Classic 普通 OAuth 绑定入口误生成 intent=login，可创建新账号并替换当前登录态。
  绑定入口明确 intent=bind，保持发起 Session；登录/注册入口保持 login，绑定不能登出，回调不得把绑定结果当登录数据。
- Classic Telegram 绑定指向已移除的无 flow 路径，应使用现有 bind/start 与一次性 callback_url，校验回调来源及 flow。
- Default 邮箱绑定发验证码未提供已启用的 Turnstile；补齐现有验证组件和 token 生命周期。
- Classic 邮箱发验证码 URL 未编码，带 + 的地址会变成空格；改用 params，修正失败后 loading/按钮和成功后用户信息回读。
- Classic 个人设置将 `/api/user/self` 的纯资料覆盖本地认证数据，丢失原令牌与 Session。资料回读改为校验用户 ID 及 Session 未切换，再合并最新凭证；邮箱成功后回读并同步 Context 和本地数据，不再直接修改 Context 对象。

## 契约与安全边界

不新增自动合并账号，不凭相同邮箱绑定第三方账号，不移动用户额度/令牌/记录，不对线上账户执行修复写入。
邮箱成功响应和 OAuth flow 接口保持，普通用户只能绑定到自己的当前身份；OAuth 仍使用服务端绑定发起用户与 Session 的一次性 state。
绑定失败不能进入注册兜底或接受新的登录 bundle。用户已有会话、密码、余额、角色、分组与社交 ID 不由邮箱操作改写。
本次不改数据 schema，不调用真实邮件/社交提供商，生产仅只读取证。

验证码存储仍沿用进程内实现，消费原子性作用于接收该验证码的进程；没有在本次引入跨实例验证码共享。
多实例需要沿用现有会话路由约束。用户注册和密码重置的验证码调用未更改，不能将本次绑定修复描述成认证系统全面改造。

## 验证计划

后端覆盖原用户原位绑定、用户数量不增、用户不存在或删除时不插入、并发变更不被旧快照覆盖、邮箱占用拒绝、验证码错误和重用拒绝、真实 UserAuth 路由归属。
Classic 覆盖所有普通 OAuth 入口 bind intent、登录保持 login、绑定不登出、回调身份保持、Telegram flow，以及邮箱编码和失败恢复；Default 覆盖 Turnstile 开启/关闭/重试及邮箱原有提交契约。
每个单元测试进程上限 60 秒，执行相关 lint、Default 类型检查、两模板构建及文档检查。

## 后端回归证据

- 修复前：`TestBindEmailPreservesConcurrentUserChanges` 复现旧用户名、密码、角色、启用状态和分组写回，并意外改变 AuthVersion；`TestEmailBindRouteKeepsAuthenticatedAccount` 复现验证码可重复使用。
- 修复后：目标测试验证仅 email 改变、规范化保留 `+`、已占用邮箱拒绝、不存在/删除的用户不会被插入、错误验证码和未认证请求不修改数据、成功绑定不签发新 Cookie/登录包。
- 真实路由测试在独立 SQLite 数据库和 Session 下执行；绑定前后用户数与 Session 数不增加，原访问令牌访问 `/api/user/self` 仍返回原 ID、资金及新邮箱。MySQL/PostgreSQL 复用现有邮件锁和 `lockForUpdate`，本次尚未运行真实数据库集成。

## 验证结果

- `go test ./common ./model ./controller ./router -count=1 -timeout 60s`：四个包全部通过。
- `go vet ./common ./model ./controller ./router`、`go build ./...`：通过；后端独立只读审查未发现阻断问题。
- Default 邮箱真实交互：修复前 7 失败 / 2 通过，修复后 9/9 通过；类型检查、局部 lint、格式与生产构建通过。
- Classic 新增 29 项实际交互、既有 10 项刷新回归和 9 项认证契约通过；包括冷启动 Context 未就绪、同 Session 令牌刷新保留、跨账号迟到响应拒绝、邮箱参数与 Turnstile 重试、五类 OAuth 入口及 Telegram flow。局部 ESLint、Prettier、生产构建和独立源码复审通过。
- 新文档索引链接、Markdown 格式与 `git diff --check`：通过。

本轮验证未向真实邮箱或社交提供商发请求，未发布版本或操作线上账号。完整邮件投递、第三方授权和历史用户现场仍需与已证实的本地行为区分。
