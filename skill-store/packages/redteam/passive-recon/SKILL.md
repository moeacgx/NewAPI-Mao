---
name: passive-recon
description: 被动信息收集：不接触目标主机，仅使用公开数据源
---



## 数据源
- 证书透明日志（crt.sh）：`curl -s "https://crt.sh/?q=%25.<domain>&output=json" | jq -r '.[].name_value' | sort -u`
- DNS 记录：`dig +short <domain> A/AAAA/MX/TXT/NS`
- whois：注册人、网段、ASN
- 搜索引擎与代码托管平台：泄露的子域、密钥、内部地址
- 公开端口测绘数据（Shodan/Censys/FOFA 等，需授权 API）

## 记录（建议写入本地 JSON 或 SQLite）要求
每条发现调用 记录或更新资产到本地清单，provenance 固定为 `passive`，tool 记录具体数据源。


