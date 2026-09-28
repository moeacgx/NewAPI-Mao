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
- 没有取得上游原始响应正文，因此不能宣称生产响应使用了某一种图片载体。

## 源码根因

固定基线 `v1.0.0-rc.10.1.10.341`（`7f6a15879`）中：

- 图片请求会命中 Gemini 原生生图模型判断，转换为
  `contents[].parts[].text` 和 `responseModalities=[TEXT,IMAGE]`；自动尺寸、自动质量
  不产生 `imageConfig`。
- 支持的 Gemini 生图模型由 `GeminiNativeImageHandler` 解析，不走 Imagen 的
  `predictions[]` 处理器。
- `GeminiNativeImageHandler` 只读取 `parts[].inlineData.data`，忽略 DTO 已定义的
  `parts[].fileData.fileUri`。
- DTO 可读取 `inline_data` / `mime_type`，但未读取等价的
  `file_data` / `file_uri`。

因此，`fileData` 是可用最小 fixture 明确证明的兼容缺口；真实生产回包载体仍需
后续脱敏响应形状证据确认。

## 修复契约

- `inlineData` 与 `inline_data` 的非空 `data` 输出为 OpenAI `b64_json`。
- `fileData` 与 `file_data` 仅在 `fileUri` / `file_uri` 是 host 有效的 HTTP(S)
  地址，且 MIME 为空或明确为 `image/*` 时输出为 OpenAI `url`；网关不主动下载
  远程图片。`gs:`、`file:`、`javascript:`、空 host 和显式非图片 MIME 均拒绝。
- 文本 part、空媒体 part、工具 part 和安全拒绝不得伪装成成功图片。
- 无图片时继续返回 `bad_response_body`，同时只记录候选数、part 类型计数和
  固定白名单内的 finish reason / block reason；未知值统一记录为 `OTHER`。
  诊断通过 `logger.LogWarn` 写入带 request ID 的运行日志，不写入数据库消费日志，
  且不得包含文本、URL、base64 或鉴权信息。
- 保持 OpenAI Images 与 Canvas 任务协议，不把免 Key 请求改走原生 Gemini 路径。
- `thought=true` 的非空 `inlineData` 延续既有行为，仍作为图片输出；本次不改变
  Gemini 思考 part 的历史处理语义。

## 测试计划

1. 使用原始 JSON fixture 覆盖 camelCase/snake_case 的 inline/file 媒体载体。
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
