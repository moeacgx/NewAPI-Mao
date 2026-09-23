# TypeSafe 原生路由回归样例

`typesafe-1.0.0.js` 为未修改的官方插件源码，来源为
[QuantumNous/new-api-plugins 的固定版本](https://github.com/QuantumNous/new-api-plugins/blob/b42cc99a6bd1998d0cc1581270bd46798ce00ad6/plugins/tasks/typesafe/1.0.0/plugin.js)。

作者与元数据保留 QuantumNous；许可证见 `typesafe-LICENSE`（Apache-2.0）。
SHA-256：`80585e402c8e6709f976e6be1f0308a95d3b991370383ab984d21fe968d8e912`。

集成测试将请求发送至本地模拟 TypeSafe 服务，验证网关鉴权、路由、账务和同步结果契约。
这些测试不访问 TypeSafe，也不能作为真实供应商或生产验收证明。
