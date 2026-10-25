# .354 技能更新、删除与大仓库导入发布

## 目标与范围

用户于 2026-10-25 要求将技能中心近期修复逐节点更新到正式站 maolaoapi。正式站使用发布的
固定 GHCR 镜像，因此先发布 `.354`，再按[正式站部署记录](25_maolaoapi_354_deployment.md)
逐节点更新。

`.353` 之后合入 `custom-main` 的变更：

- #329：技能记录 GitHub 来源并支持一键更新、管理员删除（自动卸载令牌安装），用途说明取消
  512 字符限制；Default 与 Classic 双模板。记录见
  [GitHub 技能来源、一键更新与删除](25_github_skill_updates.md)。
- #331：GitHub 大仓库导入改为有限并发下载、期限 90 秒、瞬时错误有限重试、NUL 转写。
  记录见 [GitHub 大仓库技能导入超时](25_github_skill_import_concurrency.md)。
- #333：Default 技能编辑弹窗引用正文默认折叠、修复长行横向溢出、保存栏固定。
  记录见[技能编辑弹窗引用正文折叠](25_skill_editor_reference_collapse.md)。
- #330、#332：zzapi 部署记录，仅文档。

`relaykit/`、Go 依赖与前端依赖清单未改变。

## 兼容与回滚边界

- 唯一数据库变更为 `prompt_templates.repository_source_json`（可空 TEXT），由 AutoMigrate
  新增；不修改、不删除既有列。
- 滚动期间 `.353` 节点忽略新增列，读取与保存模板不受影响；`.353` 保存模板使用显式列映射
  `Updates`，不含 `repository_source_json`，因此不会清空已记录的来源。
- 删除技能与 GitHub 一键更新路由只在 `.354` 节点存在；滚动期间请求落到旧节点时进入
  `NoRoute` 返回 404，不产生部分删除或部分更新。
- 回滚可退回 `.353` 固定镜像，新增列无需删除；回滚后 GitHub 一键更新与删除入口消失，
  已删除的技能不会恢复，如需恢复只能从部署前数据库备份单独提取。
- 不改鉴权、计费、模型路由与跨项目契约。

## 发布门禁

- 发布 PR 与合并后 CI 通过后，在合并提交创建 `v1.0.0-rc.10.1.10.354` 附注标签。
- 验证 Release 的 Linux 资产与 `checksums-linux.txt`、GHCR 固定镜像架构、
  `org.opencontainers.image.version` 与 `org.opencontainers.image.revision`，再设为 Latest。
- Classic 不在 CI 前端任务中，发布前在本分支本地执行提示词中心测试、翻译检查与生产构建。

## 验证结果

- 发布分支（基于 `705ce40ad`）本地 Classic：`vitest run src/pages/PromptCenter/__tests__/`
  5 文件 62 项通过，`node --test i18n.test.mjs` 通过，`vite build` 成功。
- Default 与后端由 #329、#331、#333 各自合并前 CI 及本地回归覆盖：Default 提示词中心
  81 项、`tsgo -b`、`rsbuild build`；后端 `model`、`service`、`controller`、`router` 测试与
  `go vet`。
- 未运行真实 MySQL/PostgreSQL 实库回归；正式站迁移由部署记录中的主节点迁移验收确认。
