# 上游认证差异与安全兼容补丁

## 目标与范围

交付基线为 `origin/custom-main@45ab82100a74b9e7c3866f1c173d16d32745527e`，工作分支为 `agent/upstream-auth-telegram`，PR 目标为 `custom-main`。固定上游分析点 `9fe0457ee`，rc.37 参考为 `385d2dfd1`，不跟随浮动上游。本工作核对 Telegram、统一登录、安全验证、Passkey、密码存储、个人访问令牌和审计，保留本地服务端 Session、AuthVersion、SessionID、SessionVersion、刷新轮换和撤销栅栏。

首次错误基于旧本地 `353352428`。同步前落后 92 个提交，独有 `353352428`、`99db8805c`、`5fb6a75ca` 三个合并提交，相对共同祖先 `cc8fd2bbe` 没有文件差异。已保留 `backup/upstream-auth-before-sync-20260915` 和包含未跟踪文件的 stash `d644518c0210d1a5ae4587fc741c236a7b2afdf3`，将本分支基于最新远端后恢复补丁。仅开发文档索引发生冲突，按条目保留两侧新增内容；未带入旧合并提交或其他业务差异。

本阶段交付认证后端兼容补丁。Classic 的完整安全中心实际实现由协调计划中的 Classic 任务负责，本 Agent 提供准确 API 矩阵；本 PR 不修改两套前端，也不宣称 Classic 全量适配完成。现有旧 GET/登录仍可继续使用，未配套之前不切换 OAuth/410。

上游参考不是单一提交：`d8cb17744` 为访问令牌/审计，`45c3fbe8a` 为操作与会话绑定 proof，`3e84ec0ab` 为 Telegram OAuth，`0973dc2b8` 为 Argon2id 与账户安全，`6f2333990` 为统一登录验证。不得将后续能力归因于 Telegram 提交。

## 本轮方案与安全边界

- Telegram 绑定 flow 固定发起时用户和会话版本，回调与绑定事务重复检查；无版本的历史在途 flow 失败关闭，用户重新发起即可。保留签名时效、断言一次性消费、唯一归属和事务回滚。
- Passkey 注册和 step-up 的 flow 固定四元身份，并在消费事务中校验数据库状态；注册最终凭据写入与身份复核、AuthVersion 推进处于同一事务。未登录 Passkey 登录仍使用零身份，其 WebAuthn 协议不变。
- 统一登录出口使用主凭证认证时读取的 AuthVersion，防止凭证验证与会话签发之间发生密码重置后仍获新会话。
- Argon2id 仅增加上游固定参数的有界读取；原 bcrypt 写入和备份码算法保持现行契约，不启用新密码长度策略或自动重哈希。
- PAT 增加 POST 生成和 DELETE 撤销；保留旧 GET 生成。撤销锁定实际代际，只修改 PAT，不推进用户/浏览器会话版本。生成/撤销记录指纹，不记录原文。
- 审计继续写本地 logs，保留 channel.update 过滤及现有权限投影。未迁移独立 audit_logs 或 last_used_at/last_used_ip 聚合。

## 测试计划

先复现旧凭据快照签发、Telegram 同 SID 版本推进及 Argon2id 读取失败；随后测试 PAT 旋转/撤销幂等、脱敏审计和浏览器会话不受影响。运行 controller、middleware、service、model、common 的认证定向测试，每个 Go 测试进程超时 60 秒。

## 验证结果（2026-09-15）

以下检查在本分支本地通过：

```sh
go test ./common ./controller ./middleware ./model ./service ./service/passkey -run 'Test.*(Auth|Session|Telegram|Passkey|Password|AccessToken|PAT|TwoFA|TwoFactor|Proof|Login|Audit|ExternalIdentity|SecurityFactor)' -count=1 -timeout=60s
go test ./controller -run 'Test(TelegramBindStartFreezesIdentityAndRejectsUnversionedFlow|PasskeyEnrollmentRejectsRevokedSessionAtWrite|RevokeAccessTokenIsIdempotentAndPreservesSession)$' -count=1 -timeout=60s
go test ./router -run '^$' -timeout=60s
go vet ./common ./controller ./middleware ./model ./service ./service/passkey
git diff --check
```

- 六个认证相关包通过；路由检查仅证明编译，不是 HTTP 全路由集成测试。
- 新增回归曾在修复前失败，覆盖旧认证快照签发、Telegram 两种版本推进、Argon2id、PAT 缺审计，以及 Passkey flow/最终写入的撤销竞态；修复后通过。另验证旧空 payload 失败关闭、PAT 撤销幂等与会话保留。
- 现有定向测试继续覆盖内部 JWT 不降级 PAT、AuthVersion/SessionVersion、刷新重放、原子准入/旧会话淘汰、Redis deny fence，以及 channel.update 审计抑制。
- Go 文件已 gofmt；四份变更 Markdown 使用仓库依赖同版本 `oxfmt@0.57.0` 格式化，新增/修改的三条本地文档链接检查通过。
- 未运行前端构建或真实浏览器/Passkey 硬件验收；前端兼容性依据源码契约。未请求 Telegram OIDC/JWKS、未测试正式 client ID/secret。
- 本地数据库回归使用 SQLite，Redis 行为由现有测试夹具覆盖；MySQL/PostgreSQL 锁语义依据既有 `lockForUpdate` 实现静态核对，未运行这两类真实数据库。
- 最新基线重新运行六包认证定向测试全部通过：common 1.447s、controller 2.682s、middleware 13.499s、model 10.071s、service 13.827s、service/passkey 0.212s；定向 vet 和路由编译再次通过。保留基线新增加的直登会话 30 天票据配置，不改变 JWT 的普通 15 分钟有效期。
- 未修改 relaykit 或其公共 API，未触发独立模块构建要求；未部署、未执行生产操作。本阶段创建边界明确的草稿 PR，完整安全中心、Classic/Default 配套与人工验收仍未就绪。

## Telegram 3e84ec0ab 的实际契约

上游明确将以下旧入口退役为 HTTP 410，错误码为 `TELEGRAM_LEGACY_AUTH_REMOVED`：

- `GET /api/oauth/telegram/login`
- `POST /api/oauth/telegram/bind/start`
- `GET /api/oauth/telegram/bind/:flow_token`

新路径通过 `POST /api/oauth/state` 返回 `authorization_url`；使用授权码、S256 PKCE 和经过签名、issuer、audience、有效期校验的 ID Token。服务端保存 verifier；Token 交换使用 client ID/secret 的 HTTP Basic 认证。凭据配置为 `telegram.client_id`、`telegram.client_secret`，**现有 TelegramBotToken/TelegramBotName 不能替代，也不得静默兜底**。回调地址由 `ServerAddress` 得到 `/oauth/telegram`；状态含 `telegram_oauth_configured`。自定义 OAuth 的 `telegram` slug 必须让位于内建提供者，存量冲突应失败关闭。

现有数字 `users.telegram_id` 和身份归属表继续使用；未绑定 Telegram 账号不能落入普通 OAuth 自动注册流程。绑定除了校验 state，还必须固定发起者的完整 Session 身份，并在绑定事务内重验有效性。不能直接覆盖为普通用户字段更新。

本分支仍运行 Widget 协议，**没有实施 410、OAuth provider 或新配置**。原因是 Default 和 Classic 尚未迁移，且该上游提交依赖本地未引入的统一验证、OIDC 库及前端授权流程。本轮完成差异审查与可兼容安全加固，不声明整个上游系列已合入。

## 仍需整体迁移的安全差异

| 能力                | 本分支当前状态                                                                                   | 上游要求与剩余工作                                                                                                                        |
| ------------------- | ------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| 统一登录验证        | 密码入口使用 `require_2fa`，其他主认证入口未统一选择第二因子；签发出口本轮固定认证时 AuthVersion | `6f2333990` 的 `require_verification + methods`、TOTP/Passkey 选择、challenge 原子消费与会话签发，需要两套 UI 同步                        |
| session-bound proof | JWT 已绑定 SID/uv/sv/method/scope；五分钟内可复用                                                | `45c3fbe8a` 的操作 context_hash、一分钟单次 proof、数据库消费及失败不恢复；不能只改 TTL 或仅保护 begin                                    |
| 首次安全因子配置    | Passkey flow 与最终写入已冻结身份；首次注册的方法策略和 2FA pending 状态仍沿用本地               | 完整迁移统一方法策略、session-bound 2FA enrollment，已有密码或因子要求再次验证；不得将裸会话作为通用兜底                                  |
| Passkey 用户验证    | 保留本地 preferred/配置行为                                                                      | 上游统一强制 UV required、兼容 RP 域名与旧 challenge 拒绝，需要实际设备与两套页面验证                                                     |
| Argon2id            | 读取固定 `v=19,m=19456,t=2,p=1`、16 字节盐、32 字节输出；错误参数在 KDF 前拒绝；仍写 bcrypt      | 账户写入和 8–128 字符策略、加密传输 v2 尚未迁移；备份码继续 bcrypt                                                                        |
| PAT                 | GET/POST 生成新明文一次返回；DELETE 幂等撤销；日志仅 SHA-256 代际指纹                            | 尚无 token/status、created_at、last_used_at/last_used_ip；上游新安全页不能直接接入                                                        |
| 审计                | 本地 LogTypeManage 与登录日志仍落 logs；补生成/撤销操作                                          | 独立 audit_logs、PAT 请求成功/失败覆盖、角色快照与筛选 API/UI 未迁移；必须保留 channel.update 精确抑制，JSON 列不能原样照搬上游 type:json |

## 前后端兼容矩阵

本轮未修改任何前端。下表基于源码调用契约核对，不等于真实浏览器验收。

| 客户端                | 原本地后端                                                         | 本轮兼容补丁后端                                                            | 完整上游迁移后端                                    |
| --------------------- | ------------------------------------------------------------------ | --------------------------------------------------------------------------- | --------------------------------------------------- |
| 当前 Default Telegram | Widget 登录与 flow 绑定                                            | 登录/绑定响应结构不变；在途旧绑定需重启                                     | 旧入口 410，必须迁移 authorization_url 和回调       |
| 当前 Classic Telegram | Widget 登录；绑定仍调用未注册的无 token `/api/oauth/telegram/bind` | 登录兼容；原绑定问题仍存在，本轮未修改 Classic                              | 登录 410；须独立迁移 Classic，不能依赖 Default 修改 |
| 当前两模板 Passkey    | flow_token + credential、scope proof                               | 请求/响应不变；旧注册/step-up flow 需重新发起                               | context、单次 proof、enrollment、UV 策略须同步升级  |
| 当前两模板 PAT/日志   | GET `/api/user/token`，读取 logs                                   | GET 保留；新增 POST/DELETE；新增审计保留原 Other 字符串结构与 Content 兜底  | 独立审计表切换后旧日志页缺少新事件，需新审计页      |
| 上游新安全页          | 缺统一验证和 PAT 状态 API                                          | 仍不能整体接入                                                              | 需同版本后端及配置                                  |
| 自建会话客户端        | Bearer JWT + Refresh Cookie                                        | AuthBundle、sid/uv/sv、refresh/logout 不变；PAT 不能成为 Session proof 身份 | 必须核对统一 challenge、操作 context 和一次性 proof |

取证入口：Default `web/src/features/auth/hooks/use-oauth-login.ts`、`features/profile/components/tabs/account-bindings-tab.tsx`、`features/profile/api.ts`；Classic `web/classic/src/components/auth/LoginForm.jsx`、`RegisterForm.jsx`、`components/settings/personal/cards/AccountManagement.jsx`、`components/settings/PersonalSetting.jsx`。

## 配置迁移和滚动发布

### 本轮补丁

1. 无新增数据库列/表，无新环境变量，不修改 `SESSION_SECRET`、Redis 拓扑或会话期限。
2. 所有节点一起升级或短窗口滚动更新；旧节点仍可接受旧流程，因此只有所有节点完成升级后，版本冻结防护才完整生效。
3. 升级时现有登录 Session、Refresh Cookie 和 PAT 继续有效。旧 Telegram bind payload 没有身份，返回 `TELEGRAM_BIND_FLOW_INVALID`；旧 Passkey 注册/step-up payload 返回流程失效。重新开始操作即可，不需要清表。
4. 本轮没有 `ACCOUNT_PASSWORD_HASH_ALGORITHM` 写入开关。不要通过设置它误以为本分支已经启用 Argon2id；新注册、密码修改、备份码仍走现有 bcrypt helper。
5. PAT POST/DELETE 使用当前 UserAuth 权限和禁缓存策略；POST 与旧 GET 共享生成限流桶。DELETE 返回 `{success:true,message:"",data:null}`，重复删除不重复记录撤销；只撤销操作者自己的 PAT。

### 未来完整上游迁移

1. 先准备独立 Telegram client ID/secret、准确 HTTPS 回调和备用登录方式；处理自定义 telegram slug 冲突。Bot Token 可为回滚保留，但不得用于新 OAuth 认证。
2. 配套发布后端及两套前端，验证已有绑定、未绑定拒绝、撤销/版本变化、PKCE/ID Token 错误和 410，然后启用 Telegram。密钥通过现有配置通道保存，不写日志或文档。
3. Argon2id 分两阶段：先全部节点具备双格式读取；完整写入实现上线时使用 `ACCOUNT_PASSWORD_HASH_ALGORITHM=bcrypt` 过渡，再统一启用 argon2id。禁止批量重写历史哈希。每次 KDF 约使用 19 MiB 内存，需要针对实际登录并发评估资源。
4. 新审计存储要支持 SQLite/MySQL/PostgreSQL，采用 TEXT 兼容 JSON 内容；分别验证日志库和主库，不能将消费日志清理直接作用于审计表。

## 回滚

- 本轮可回滚代码，不删用户、会话、AuthFlow、身份归属或日志数据；仍需保留相同 SessionSecret。回滚将重新暴露旧版本流程检查缺口，应视为安全能力退回。
- 新旧在途流程不能假定可互用；让用户重新开始绑定和 Passkey 操作，禁止去掉会话检查来兼容旧 payload。
- PAT 撤销/旋转不会被代码回滚撤销，旧令牌不能恢复使用；审计记录继续保留。客户端若使用新增 POST/DELETE，回滚后需回到对应旧功能入口，不能用 GET 生成模拟撤销。
- 本轮不会产生 Argon2id 哈希；若数据库被其他版本写入 Argon2id，禁止回滚到 bcrypt-only reader，否则对应用户无法登录。应保留双格式 reader 并选择前向修复。已启用上游密码传输 v2 时，同样必须保留 v2 reader 或配套回滚前端。
- 完整 Telegram OAuth 切换后须成套回滚前后端与配置，保留数字 telegram_id 及外部身份归属，旧 Bot 凭据不能替代新 client_secret。

## 跨项目影响

仅确认新增面板 PAT 的 POST/DELETE 能力；模型中继、API key、计费用量及 tokens-pro/sub2api 共享数据没有变更。未发送跨项目 Issue。会话四元身份转为 model 层共享类型，service.AuthIdentity 使用 alias；JWT claim 名 sid/uv/sv 和校验链保持一致。
