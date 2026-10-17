# 容器资源过载保护

## 目标

修正容器部署中系统过载哨兵读取宿主机 CPU/内存口径、瞬时尖峰立即拒绝 Relay 请求的问题。保护动作仍返回 HTTP 503，并保留既有错误码：`system_cpu_overloaded`、`system_memory_overloaded` 和 `system_disk_overloaded`。

## 变更范围

- CPU 优先读取 cgroup v2 的 `cpu.max` / `cpu.stat`，兼容 cgroup v1 的 CPU 配额和累计用量；按容器配额与可调度 CPU 数计算占用率。
- 内存优先读取 cgroup v2 的 `memory.current` / `memory.max` / `memory.stat`，兼容 cgroup v1；使用工作集（扣除非活跃文件缓存）计算占用率。
- cgroup 未限额、文件不可读或工作集字段不完整时，回退到原有宿主机口径，保持非容器环境兼容。
- CPU、内存和磁盘各自需要连续 3 次采样超过当前阈值才标记过载。监控采样间隔约 5 秒，因此约 15 秒后才会拒绝请求；任一采样恢复到阈值以下即清零。
- 采样超过 30 秒未刷新时不再触发拒绝，避免监控暂停后使用陈旧状态熔断流量。运行时关闭监控或提高阈值后，当前请求检查立即重新按配置放行。

## 兼容性与安全边界

- 不改变性能监控配置字段、Relay 路由范围、HTTP 503 状态码或客户端错误码。
- cgroup 数据只用于过载保护，不写入数据库，也不改变计费、重试和流式响应处理。
- 本次变更不等同于修复客户端或 Cloudflare 提前断开造成的 HTTP 499；它只减少资源瞬时尖峰导致的 503 拒绝。

## 验证

```text
go test -timeout 60s ./common/... ./middleware/...
git diff --check
```

测试覆盖 cgroup v1/v2 CPU 配额、CPU 首次采样基线、内存工作集及缓存扣除、缺失数据回退、连续采样门槛和陈旧采样放行。

## 来源

实现参考上游 `QuantumNous/new-api` 提交 `09109ec31`（2026-10-16），在 NewAPI-Mao 当前基线 `5975719f7` 上独立提交，未合并其他上游大范围性能统计重构。
