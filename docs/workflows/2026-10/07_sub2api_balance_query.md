# Sub2API 渠道余额查询

## 目标

为 Sub2API 渠道增加后台余额查询和定时余额刷新，复用其 `/v1/usage`
接口，避免把 Sub2API 当作普通 OpenAI 渠道访问不存在的计费接口。

## 行为契约

- 请求 `GET {base_url}/v1/usage`，认证方式为渠道密钥对应的 Bearer 头。
- 响应同时存在 `remaining` 和 `balance` 时优先使用 `remaining`；只有
  `remaining` 缺失时才回退到 `balance`。
- `isValid=false` 视为密钥失效，余额不更新。
- 响应缺少两个余额字段时返回错误，余额不更新。
- 无限制订阅可能返回 `remaining=-1`。当响应明确包含没有日、周、月限额的
  subscription 时，该值视为无限制标记，不写入渠道余额。
- 普通钱包允许真实负余额，不能仅凭 `-1` 一律拒绝。

## 兼容性与安全边界

只新增 Sub2API 类型分支，不改变其他渠道的余额接口和余额单位。请求继续复用
现有 `GetResponseBody` 和代理设置，密钥只放在 Authorization 请求头，不写入
错误响应或文档示例。

## 验证

```text
go test -timeout 60s ./controller -run 'TestUpdate(Channel|Standard).*Sub2API'
git diff --check
```

测试覆盖 `remaining` 优先、钱包回退、无效密钥、无限制订阅和统一余额路由。
