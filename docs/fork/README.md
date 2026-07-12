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

## Agent 技能（逐步教练）

仓库内 skill：**`ggapi-fork`**（路径 [`.agents/skills/ggapi-fork/`](../../.agents/skills/ggapi-fork/)）。

在 Grok / 兼容 agent 中可用：

- 斜杠命令：`/ggapi-fork`
- 自然语言：二开、开分支、push 还是 PR、同步上游、冲突、差异清单、发版回归等

Skill 会按模式（日常开发 / 推送与 PR / 上游同步 / 冲突与清单 / 发版 / 排障 / **skill 自升级**）逐步给出命令，并与本目录 SOP 对齐；**流程以本目录文档为准**，skill 负责执行级指引。当前 skill 版本见清单 [GG-002](./diff-inventory.md)（**v1.5.0+**）。本仓 git tag 规则见 [SOP §5.1](./branch-and-sync-sop.md)（**`v<上游基线>.N`**，例 `v1.0.0-rc.20.1`；基线须为 `origin/main` 祖先；正式 tag 只钉 `origin/main` tip；`N` 仅在基线版本串变化时归 1；**默认产物 = GHCR 镜像**，裸二进制仅手动 dispatch）。

**最终提交前：** skill Mode C（含用户直接说「提交/commit」）会先跑 Codex 审查（agent 优先用 companion/`codex review`，用户也可用 `/codex:review`），有实质问题则修复后再审（默认最多 3 轮）；须覆盖 **相对 `origin/main` 的完整最终树**（已提交 + 未提交并存时先 materialize 再审）。通过后再落最终 commit 说明；审查本身不改代码逻辑、也不自动 push。跳过须用户明确说；文档/skill 改动不默认豁免（见 skill `references/workflows.md` C2-pre / troubleshooting §21）。

自升级策略（能改进自身、禁止乱改）：见 skill 内 [`references/self-upgrade.md`](../../.agents/skills/ggapi-fork/references/self-upgrade.md)。摘要：**L1** 对齐文档/小修补可主动改文件但不自动提交；**L2/L3** 须先方案后确认；永不静默削弱 hard rules、不自动 push。

## 核心原则

1. **薄定制层**：能配置解决的不改代码；能独立扩展点解决的不改上游热点文件。
2. **可追踪差异**：凡相对 `upstream` 的永久补丁，必须登记到差异清单（含本目录与 `AGENTS.md` 二开入口本身，见 [GG-001](./diff-inventory.md)）。
3. **定期同步**：安全修复尽快合入；功能跟进按节奏 merge，避免分叉过大。
4. **高危二审**：计费、鉴权、relay 协议、数据库事务/锁相关改动必须第二人 Review。
5. **品牌与许可**：遵守 AGPLv3；不得删除或替换 new-api / QuantumNous 等受保护标识（见 `AGENTS.md` Project Governance）。
6. **`AGENTS.md` 冲突策略**：该文件上游也会改；同步时优先保留本仓「ggapi 二开维护」入口，再合入上游工程规则变更。

## 前端壳（ggapi 第三壳）

| 路径 | 角色 | 本地命令 |
|------|------|----------|
| `web/ggapi` | **本仓产品壳**（默认主题） | `make dev-web-ggapi` / `make build-web-ggapi` |
| `web/default` | 上游 default，同步用；功能源 | `make dev-web` |
| `web/classic` | 上游经典壳 | `make dev-web-classic` |

功能策略：**产品与视觉在 `web/ggapi`；功能追 `web/default`**（同步后 port，见 [SOP §3.5](./branch-and-sync-sop.md)）。清单项 [GG-005](./diff-inventory.md)。

## 快速入口

```bash
# 日常前端 + API 开发（产品壳）
make dev-api
make dev-web-ggapi
# 或：make dev   # API + ggapi 壳

# 同步上游（详见 SOP）
git fetch upstream
```