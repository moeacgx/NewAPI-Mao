# .367 充值限流隔离发布

## 目标与范围

本次版本为 `v1.0.0-rc.10.1.10.367`，修复只读令牌余额、日志查询占满共享 IP
关键操作限流后阻塞充值的问题，并统一 Default、Classic 充值、订阅、历史订单
重新支付的 HTTP 429 提示。实现及配置见[限流隔离记录](06_token_query_payment_rate_limit.md)，
Classic 界面范围见[支付限流提示](06_classic_payment_rate_limit.md)。

用户授权复核后提交 PR、合并项目主分支 `custom-main` 并发版；本次发布不包含生产实例部署。
使用独立分支，不纳入另一个尚未合并的生产部署文档 PR 或工作区未跟踪资料。

## 复核与验证

- 补齐两套模板历史订单重新支付的错误处理；Default 正确读取 AxiosHeaders 的
  `Retry-After`，登记动态翻译键；Classic 清理本轮重复版权头。
- 后端通过真实 `SetApiRouter` 路由验证查询与充值 CT 双向隔离、按用户计数、
  Redis/内存、GA、开关、认证和禁用边界；支付仅到参数校验，不调用支付商。
- 前端通过真实 Axios 拦截器与付款操作验证 HTTP 429 只弹一条局部提示，缺失等待时间
  和其他错误仍正常提示；所有语言包保持占位符。
- 执行后端相关包测试、vet/build、Default 类型/静态检查/测试/构建、Classic
  静态检查/测试/构建以及 Markdown 格式、链接和差异检查。远端 CI 为合并门禁。
- 本地后端相关包 719 项测试通过；Default 13 项、Classic 4 项行为测试通过，
  两套生产构建均通过。Classic 保留既有 Browserslist、eval 和大分块构建警告。

## 发布与回滚

验证通过后 PR 合入 `custom-main`，附注版本标签触发 Linux amd64/arm64 二进制与
GHCR 多架构镜像工作流。核验标签对应的提交、工作流、校验文件、二进制内嵌版本和镜像清单。
不覆盖历史标签，不修改其他服务、数据库或 Redis 配置。

无需数据迁移。升级后 `TOKEN_QUERY_RATE_LIMIT*` 独立控制查询限流；回滚恢复共享
CT 的旧行为。实际部署时需所有节点采用同一版本，滚动期间旧节点仍可能消耗 CT。

## 共享契约与分享评估

只读查询接口仍使用 Bearer API Key，响应与 HTTP 429/Retry-After 契约保持；计数主体
由出口 IP 调整为用户，并将余额和日志查询分开。模型、用量、价格及扣费不变。

NewAPIForDouDi：若保留同一 CT 查询路由，值得分享这一故障修复，需核对其鉴权上下文字段
与轮询需求，不能直接采用未评估的阈值。

NewAPIModifyByGang：若客户端余额查询与支付仍共桶，同样值得分享；需适配其前端模板和
错误拦截器，不要求新增未有的支付方式。本轮不创建跨项目 Issue 或 PR。

## 发布证据入口

缺陷跟踪为 [#392](https://github.com/moeacgx/NewAPI-Mao/issues/392)，关联 PR 提供
完整 CI 与合并提交。最终资产、校验和与镜像摘要记入
[本版本 Release](https://github.com/moeacgx/NewAPI-Mao/releases/tag/v1.0.0-rc.10.1.10.367)。
本文件记录发布方案与本地验证，不代表生产实例已升级。
