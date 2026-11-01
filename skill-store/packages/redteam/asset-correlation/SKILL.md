---
name: asset-correlation
description: 资产关联：域名↔IP↔C 段↔证书↔服务的图谱化
---

## 关联规则
- 域名解析 → `resolves` 边（domain → asset）
- C 段归属 → `contains` 边（segment → asset）
- 端口开放 → `exposes` 边（asset → port）
- 证书 SAN 共享 → `shares_cert` 边

## 记录（建议写入本地 JSON 或 SQLite）要求
通过 redteam_asset_link 写入 edge 表；重复关系幂等。
