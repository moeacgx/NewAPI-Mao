# 令牌提示词中心实现计划

> 代理执行：按 subagent-driven-development 拆分工作项，每项先验证失败用例，再实现和复验。

**目标：** 双模板提示词中心、按实际分组装载和令牌名称技能徽标。

**架构：** 独立模板与令牌关联表，管理 API 与令牌编辑隔离；请求按每次实际分组重新改写
原始 JSON，纳入计费。后端和两套前端使用独立 worktree，主代理负责运行时与最终整合。

**技术：** Go/Gin/GORM、React、Bun、现有两套组件与国际化工具。

**设计：** [令牌提示词中心](../../developer/token-prompt-center.md)。

## 全局约束

- 不修改主工作区已有未提交内容；基线 `origin/custom-main`，SHA `7785c19e9`。
- 只实现文本提示词；无脚本执行和生产调用，不部署、不向其他项目发送消息。
- 最大 8 安装项；正文每项 32768 字节，启用项合计 65536 字节；白名单按稳定 ID。
- 两套页面、实际调用分组、多组/auto 重试、透传和计费一致性均属本次范围。

## 工作项与接口

1. 后端管理与关联：`model/prompt_template.go`、`controller/prompt_center.go`、
   `model/main.go`、`model/token.go`、`controller/token.go`、`router/api-router.go`。
   提供 `GetActiveTokenPrompts(tokenID int, groupCode string) ([]TokenPromptContent,error)`，
   内容含 `ID/Title/Content/Version`；API 形状以设计文档为准。测试非法大小、越权、版本冲突、
   白名单、停用和批量计数，Go 单次超时 60 秒。
2. Default：`web/src/features/prompt-center/`、认证路由、侧栏、令牌徽标和七语翻译。
   先补安装交互、徽标数量/灰显/无安装隐藏测试，再实现与运行 typecheck、受影响 lint 和构建。
3. Classic：`web/classic/src/pages/PromptCenter/`、路由、侧栏、令牌桌面/手机徽标和翻译。
   先补相同用户行为测试，确保正文与长名称不挤压表格，运行受影响测试、lint 和构建。
4. 运行时：`service/token_prompt.go`、`controller/relay.go`、日志生成和 Gemini 计数。
   先写四协议正文保留、透传、白名单换组、防重复和计数测试，再接入选渠后的正文重建及补预扣。
5. 整合：逐项审查 diff，串行引入子分支提交；使用模拟上游复验实际路由和账务，独立审查，
   更新开发能力登记、文档索引和验证结果。无真实供应商和实库证据时如实记录。

## 状态

- [x] 源码可行性、双模板范围与实际调用分组语义已确认。
- [x] 技能徽标已加入设计。
- [x] 后端管理与关联。
- [x] Default 页面与徽标。
- [x] Classic 页面与徽标。
- [x] 协议装载、计费与日志。
- [x] 整合审查、验证与交付。

## 最终验收范围

后端四包全量回归、双模板交互与构建、Default 类型检查、relaykit 独立构建、本地 SQLite
管理接口及浏览器页面均已验证。钱包新增预扣采用原子余额检查，保留结算补扣原行为。
MySQL/PostgreSQL 实库、真实供应商和生产环境未实跑；没有推送、PR、合并到主线或部署。
详细证据和既有 lint/格式边界见[专题文档](../../developer/token-prompt-center.md)。

最终同步远端主线 `fdbe81d3a`（.349 发布后的基线），无冲突，重新执行后端四包、Default 61 项、
Classic 12 项交互和国际化、双模板生产构建及 relaykit 独立构建。所有改动保留在本地功能分支。
