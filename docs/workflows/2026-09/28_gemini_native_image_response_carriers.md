# Gemini 原生图片回包载体归一化

## 问题与证据边界

无限画布通过 `/canvas/v1/images/tasks` 提交 OpenAI Images 请求，后台重放到
`/canvas/v1/images/generations`。渠道为 Gemini（type 24），模型
`gemini-3-pro-image-preview` 与 `gemini-3.1-flash-image-preview` 都没有模型映射和
参数覆盖。

生产只读证据确认：

- 同一渠道的原生 Gemini `generateContent` 请求成功；
- OpenAI `/v1/images/generations` 和 Canvas 图片任务均收到上游 HTTP 200；
- 网关随后归一为 HTTP 500，`error_stage=parse`、
  `error_code=bad_response_body`、`no images generated`；
- 后续授权的真实上游探针仅记录脱敏形状：HTTP 200、1 个 candidate，parts 同时含
  `text` 与 `fileData`；`fileData` 的 URI scheme 为 HTTPS 且 host 非空，MIME 已存在且
  属于 `image/*`。该形状满足本修复的安全接受条件。
- 原先失败事件没有保存上游原始响应正文，因此不能把这次探针反推为某一条历史失败
  请求的唯一正文，也不能排除其他响应形状或其他失败边界。
- 后续对两个 preview 生图模型的脱敏探针均显示：HTTP 200、1 个 candidate、1 个
  text part、`inline_data=0`、`file_data=0`；text 中仅含 1 个 Markdown 图片，URL
  为有效 HTTP(S) host，图片扩展名为 PNG。没有记录图片 URL、正文、用户、渠道或密钥。

## 源码根因

固定基线 `v1.0.0-rc.10.1.10.341`（`7f6a15879`）中：

- 图片请求会命中 Gemini 原生生图模型判断，转换为
  `contents[].parts[].text` 和 `responseModalities=[TEXT,IMAGE]`；自动尺寸、自动质量
  不产生 `imageConfig`。
- 支持的 Gemini 生图模型由 `GeminiNativeImageHandler` 解析，不走 Imagen 的
  `predictions[]` 处理器。
- `GeminiNativeImageHandler` 只读取 `parts[].inlineData.data`，忽略 DTO 已定义的
  `parts[].fileData.fileUri`，并把 text part 仅当作普通文本统计。
- DTO 可读取 `inline_data` / `mime_type`，但未读取等价的
  `file_data` / `file_uri`。

因此，`fileData` 是可用最小 fixture 明确证明的独立兼容缺口；本次两个模型的真实
失败形状则进一步确认了 text 内 Markdown 图片载体缺口。历史失败正文不可回看，
不能据此宣称其他未观测形状不存在。

## 修复契约

- `inlineData` 与 `inline_data` 的非空 `data` 输出为 OpenAI `b64_json`。
- `fileData` 与 `file_data` 仅在 `fileUri` / `file_uri` 是 host 有效的 HTTP(S)
  地址，且 MIME 为空或明确为 `image/*` 时输出为 OpenAI `url`；网关不主动下载
  远程图片。`gs:`、`file:`、`javascript:`、空 host 和显式非图片 MIME 均拒绝。
- 没有合法 Markdown 图片的普通 text、空媒体 part、工具 part 和安全拒绝不得伪装成成功图片。
- text part 仅通过 Goldmark AST 识别标准 Markdown image 节点；代码块、代码 span、
  转义的 `![](...)`、普通 URL 和普通 Markdown link 不会被当成图片。图片目标经过
  Goldmark `util.URLEscape(..., true)` 解码 Markdown 转义/实体，保留已有 `%xx` 编码，
  再接受 HTTP(S)+非空 host；不下载远程资源。
- 无图片时继续返回 `bad_response_body`，同时只记录候选数、part 类型计数和
  固定白名单内的 finish reason / block reason；未知值统一记录为 `OTHER`。
  诊断通过 `logger.LogWarn` 写入带 request ID 的运行日志，不写入数据库消费日志，
  且不得包含文本、URL、base64 或鉴权信息。
- 保持 OpenAI Images 与 Canvas 任务协议，不把免 Key 请求改走原生 Gemini 路径。
- `thought=true` 的非空 `inlineData` 延续既有行为，仍作为图片输出；本次不改变
  Gemini 思考 part 的历史处理语义。

## 测试计划

1. 使用原始 JSON fixture 覆盖 camelCase/snake_case 的 inline/file 媒体载体及 text
   内 Markdown 图片。
2. 验证文本-only、安全结束和空媒体仍返回 `no images generated`。
3. 验证安全诊断只包含 shape 计数，不包含响应正文。
4. 验证精确 Canvas 请求仍保留 prompt、`responseModalities`，且 auto 不生成
   `imageConfig`。
5. 运行 Gemini 定向测试、根模块相关测试，以及 relaykit 独立测试和构建。

## 验证结果

隔离 worktree 使用本地工具链、无依赖下载执行：

```powershell
$env:GOARCH = "amd64"
$env:GOWORK = "off"
$env:GOPROXY = "off"
$env:GOTOOLCHAIN = "local"
go test ./relay/channel/gemini -count=1 -timeout=60s

cd relaykit
go test ./... -count=1 -timeout=60s
go build ./...
```

此外，独立临时验收程序调用真实 `GeminiNativeImageHandler` 并把输出包装成 Canvas
任务 SUCCESS 信封，再交给无限画布真实 `requestGeneration`：`inlineData` 和合法
HTTP(S) `fileData` 均返回一张图片，usage 保持不变；远程 URL 模拟 CORS 失败时仍
保留 URL；文本-only 仍返回 `no images generated`。这些均为本地 fixture，不是生产
上游响应正文证据。

本轮还生成了临时 Linux amd64 stdin 验收桥：它只允许两个 preview 模型，从 stdin
读取 Gemini 响应并输出 HTTP、错误码、图片数量、carrier 类型和 usage 摘要，不输出
URL、base64 或正文。桥接源码和二进制不进入仓库或 PR；主会话 agent 已直连同一上游
后将真实响应安全传入该 handler：两个 preview 模型均返回 1 张 URL carrier，prompt /
output / total usage 分别与上游脱敏摘要一致（Pro `13/1056/1069`，Flash
`35/1596/1631`）。这是本地修复 handler 的真实响应验收，不是线上已部署 handler，
也不是浏览器 Canvas 实点；未保存 URL、正文、用户、渠道或密钥。

## 发布准备

版本准备为 `v1.0.0-rc.10.1.10.343`。已合并的 PR #287 保留 `fileData` 载体归一化和
安全 URI/MIME 校验；本 PR 新增 text 内 Markdown 图片载体归一化和形状诊断。
`.343` tag/release 创建前已核对未占用；当前基线为 `f3ff2c483e399cb432338ecb37ab2bdc50e02541`。

本 PR 同时包含唯一功能修复和 `.343` 版本准备；等待 PR 自身前后端 CI 全部通过后，
再由维护者决定是否打 tag、构建 Release、更新线上容器。生产验证、tag、合并和部署
不属于本工作项。
