# 令牌分组标签大卡片与普通多选优先级列表

## 目标与范围

- Default 令牌创建/编辑在分组选择器上方提供与 Classic 一致的分组标签大卡片（品牌图标、名称、可选分组数量、选中态），手机 2 列，抽屉宽度足够时（`sm` 起）3 列；样式限定在令牌表单，不改动其他页面的标签筛选。
- 两套模板中，普通选择器选中 2 个及以上具体分组时，显示「Custom Auto」有序优先级列表：序号、拖拽手柄、上移/下移按钮、移除；第一项最先尝试。
- Classic 原有标签卡片保留并复验；全局 Auto、显式 Auto（`auto_groups`）编辑器行为不变。

## 契约决策（线路格式不变）

普通多选继续使用显式契约：`group=a,b`、`group_ids`、`group_mode=explicit`，不转换为 `group=auto`，不改后端、计费、鉴权。依据：

- 显式多组已按保存顺序路由：`service/channel_select.go` 中 `GetRequestTokenGroups` 返回多组时走 `selectFromOrderedGroups`，首项优先。
- `middleware/auth.go` 在显式多组上校验数量上限、`GetUserUsableGroups` 与倍率保护后写入 `ContextKeyTokenGroups`。
- 转为 `group=auto` 会丢失 `group_ids`、按组渠道限定、倍率保护（`model/group_bindings.go` 仅在 explicit 时重建绑定），并跳过独立分组互斥校验，会扩大授权，因此不采用。
- 从 2 个减到 1 个具体分组时回到直接路由（`cross_group_retry=false`）；0 个仍为继承；显式 Auto 可保留 1 个分组。

## 变更清单

Default（`web/src/features/keys/components/`）：

- 新增 `group-tag-cards.tsx`：卡片网格，`aria-pressed`，空标签隐藏，当前标签失效时回到全部。
- 新增 `selected-group-order-list.tsx`：复用 `AutoGroupOrderItem`，新增 `showPriority` 序号。
- `api-key-group-combobox.tsx`：标签可受控，新增 `hideTagFilter`。
- `api-keys-mutate-drawer.tsx`：标签状态上移，卡片置于选择器上方；2 个及以上具体分组时显示列表，排序/移除同步倍率保护、渠道限定与 `cross_group_retry`。

Classic（`web/classic/src/`）：

- `components/table/tokens/modals/EditTokenModal.jsx`：2 个及以上分组显示序号、「Custom Auto」标记和可访问的上移/下移按钮（HTML5 拖拽不支持触屏，按钮即键盘与触屏兜底）。
- `components/common/group-tags/group-tags.css`：`.token-group-priority-index`。

i18n：Default 与 Classic 全部语言文件新增 `Custom Auto`；Default 新增 `Groups are tried from top to bottom. Drag to change the priority.`、`Group priority order`；Classic 新增 `Move {{name}} up`、`Move {{name}} down`。Default `bun run i18n:sync` 退出码 0。
`Custom Auto` 的中文文案定为 `自选 Auto`（Default `zh`、Classic `zh` 与 `zh-CN`），繁体为 `自選 Auto`（两套模板 `zh-TW`），保留英文 `Auto` 一词以便与全局 `Auto` 区分；其余语言与键值不变。

## 测试与验证

新增测试：

- Default `__tests__/api-keys-mutate-drawer.test.tsx`：两项分组调整顺序后提交 `group=vip,default`、`group_ids=[22,11]`、`group_mode=explicit`、`auto_groups=[]`；移除到剩一个分组回到 `default`、`[11]`、`cross_group_retry=false`。夹具中显示名称与 code 不同。
- Classic `__tests__/token-group-priority.compat.test.jsx`：序号与按钮边界、单组无序号、旧记录 `vip,default` 打开后下移并保存为 `default,vip`、`[11,22]`。

执行结果（均在 60 秒内）：

- Default 三个测试文件共 32 项通过，退出码 0；`oxlint` 退出码 0；`bun run typecheck` 退出码 0。
- Classic 逐文件运行：新增 3 项、`selection` 25 项、`token-group-limit` 3 项、`layout` 2 项，退出码均为 0。四个文件合并运行会超过包装器的 60 秒，因此分开执行。
- 本机 Node 为 v26.13.0，内置 Web Storage 会遮蔽 jsdom 的 `localStorage`，导致既有用例在 `afterEach` 报错；本次以 `NODE_OPTIONS=--no-experimental-webstorage` 仅对测试进程关闭，未修改任何测试配置。

## 未验证与已知限制

- 未做真实浏览器验证：卡片 2/3 列响应式、鼠标与触屏拖拽、键盘焦点顺序、保存后重新打开的完整回读。
- Default 旧记录打开时的顺序依赖 `resolveApiKeyGroups`（优先 `group_details`，否则按 `group` 拆分），本次未新增针对它的独立用例。
- 未构建、未提交、未部署；未跑后端测试（无后端变更）。

## 复审补正（2026-11-18 晚）

- 独立复审判定主链路通过、回归门禁未过：`group-tag-submission.test.tsx` 两处仍按浮窗内按钮查找旧标签名，
  卡片移到浮窗上方后停在 `findByRole`，后续稳定 ID/code、非管理员提交与不调用管理端接口的断言未执行。
- 已改为在浮窗外按带数量语义的名称选中卡片，再打开被筛选的选择器；保留原有稳定 ID/code、
  非管理员提交与 `/api/group/tags` 不调用的断言，并新增切换标签后已选分组（`标准当前名称`）保持的断言。
  点击卡片会关闭浮窗，因此重新打开选择器必须放在卡片点击之后。
- `docs/developer/group-tags.md` 删除了「Default 标签按钮位于选组浮窗内」的过时表述。
- 验证：Default 9 个聚焦文件 46 项通过，`oxlint` 0，`scripts/format-with-protected-headers.mjs --check` 0，
  `git diff --check` 0。生产代码未改动。
