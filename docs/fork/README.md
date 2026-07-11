# ggapi 二开维护文档

本目录约定 **AkumaRealLabs/ggapi** 相对上游 **QuantumNous/new-api** 的长期二开与同步流程。

工程代码规范（JSON、三库兼容、计费安全、前端 i18n 等）以仓库根目录 [`AGENTS.md`](../../AGENTS.md) 为准；本目录只补充 **fork 治理** 相关文档。

## 远程约定

| 远程名 | 仓库 | 用途 |
|--------|------|------|
| `origin` | `AkumaRealLabs/ggapi` | 团队主仓；**仅向此推送** |
| `upstream` | `QuantumNous/new-api` | 上游源码；**只 fetch，不 push** |

```bash
git remote -v
# origin    https://github.com/AkumaRealLabs/ggapi.git (fetch/push)
# upstream  https://github.com/QuantumNous/new-api.git (fetch)
```

## 文档列表

| 文档 | 何时阅读 |
|------|----------|
| [差异清单模板](./diff-inventory.md) | 引入/修改/废弃相对上游的定制时；每次上游同步后复核 |
| [分支命名与同步 SOP](./branch-and-sync-sop.md) | 日常开分支、合入 main、同步 upstream、发版回归 |

## 核心原则

1. **薄定制层**：能配置解决的不改代码；能独立扩展点解决的不改上游热点文件。
2. **可追踪差异**：凡相对 `upstream` 的永久补丁，必须登记到差异清单（含本目录与 `AGENTS.md` 二开入口本身，见 [GG-001](./diff-inventory.md)）。
3. **定期同步**：安全修复尽快合入；功能跟进按节奏 merge，避免分叉过大。
4. **高危二审**：计费、鉴权、relay 协议、数据库事务/锁相关改动必须第二人 Review。
5. **品牌与许可**：遵守 AGPLv3；不得删除或替换 new-api / QuantumNous 等受保护标识（见 `AGENTS.md` Project Governance）。
6. **`AGENTS.md` 冲突策略**：该文件上游也会改；同步时优先保留本仓「ggapi 二开维护」入口，再合入上游工程规则变更。

## 快速入口

```bash
# 日常前端 + API 开发
make dev-api
make dev-web

# 同步上游（详见 SOP）
git fetch upstream
```
