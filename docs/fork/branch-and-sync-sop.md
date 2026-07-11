# 分支命名与上游同步 SOP

面向 **ggapi** 长期二开：日常开发、保护 `main`、定期合并 **upstream（QuantumNous/new-api）**、发版回归。

差异登记见 [差异清单](./diff-inventory.md)；代码规范见 [`AGENTS.md`](../../AGENTS.md)。

---

## 1. 远程与分支角色

| 名称 | 角色 |
|------|------|
| `origin` | 团队仓 `AkumaRealLabs/ggapi`，唯一推送目标 |
| `upstream` | `QuantumNous/new-api`，只读拉取 |
| `main` | 可部署主干；**禁止直推**（经 PR 合入） |
| 功能/修复分支 | 短生命周期，合入后删除 |
| `sync/upstream-YYYYMMDD` | 仅用于合并上游，**禁止夹带业务功能** |

```bash
# 首次配置（若尚未添加 upstream）
git remote add upstream https://github.com/QuantumNous/new-api.git
# 建议禁止误推上游（若支持）
git remote set-url --push upstream DISABLED
```

---

## 2. 分支命名

| 前缀 | 用途 | 示例 |
|------|------|------|
| `feat/` | 新功能 | `feat/wallet-coupon` |
| `fix/` | 缺陷修复 | `fix/quota-preconsume` |
| `chore/` | 依赖、杂项、工具 | `chore/bump-deps` |
| `docs/` | 仅文档 | `docs/fork-sop` |
| `hotfix/` | 生产紧急修复 | `hotfix/login-500` |
| `sync/upstream-YYYYMMDD` | 同步上游 | `sync/upstream-20260711` |

规则：

1. 一律从**最新** `origin/main` 拉出。
2. 名称用小写、短横线，避免无意义的 `update` / `tmp`。
3. 一个分支只做一类事；同步分支不混功能。
4. 合入 `main` 后删除远程与本地功能分支。

```bash
git fetch origin
git checkout main
git pull origin main
git checkout -b feat/your-topic
```

---

## 3. 日常开发闭环

```
Issue/任务 → 分支 → 实现 → 本地验证 → PR → Review → 合入 main → 更新差异清单（如有）
```

### 3.1 实现约束（摘要）

完整规则以 `AGENTS.md` 为准，二开时特别注意：

- 分层：`router → controller → service → model`
- 定制优先落在**独立目录/包**或扩展点（新 channel、新 oauth、前端 feature）
- 少改热点：`service/*quota*`、`model/` 锁与事务、`relay/` 核心、鉴权 middleware
- 计费改动先读 `pkg/billingexpr/expr.md`，并走完整预扣 → 结算 → 退款链路自检
- 数据库改动必须 SQLite / MySQL / PostgreSQL 三端可接受
- 前端（`web/default`）：Bun、i18n、`typecheck` / lint 通过

### 3.2 本地验证（按改动范围）

```bash
# 后端：按包测试，勿整仓无脑跑除非有必要
go test ./service/...
go test ./model/...
go test ./relay/...

# 前端
cd web/default && bun run typecheck
# 以及项目约定的 lint 脚本

# 本地联调
make dev-api    # Docker 开发 API 栈
make dev-web    # 默认前端
```

### 3.3 Review 要求

| 变更类型 | 要求 |
|----------|------|
| 普通文档/独立 low 风险文件 | 至少 1 人 Review（或团队约定的自合规则） |
| 计费、额度、预扣/结算 | **强制二审** + 写清回归步骤 |
| 鉴权、Token、权限 | **强制二审** |
| `relay/` 协议与渠道适配 | 二审；说明是否影响 StreamOptions / 零值字段 |
| `model/` 迁移、行锁、事务 | 二审；说明三库行为 |
| 相对 upstream 的新 `patch` | 必须同步更新 [差异清单](./diff-inventory.md) |

### 3.4 PR 说明

- 写清：做了什么、为什么、如何验证
- 若改动是相对上游的永久差异，PR 中注明「已更新 / 将更新 diff-inventory」
- 向**官方**提 PR 时，使用 `.github/PULL_REQUEST_TEMPLATE.md`，摘要与验证步骤须人工撰写，遵循官方模板与审阅习惯

---

## 4. 上游同步 SOP

### 4.1 节奏建议

| 频率 | 动作 |
|------|------|
| 每周 | `git fetch upstream`，浏览 release / 重要 commit |
| 安全修复发布后 | 尽快开 `sync/upstream-*` 合入 |
| 常规功能 | 按双周或月度窗口 merge，避免与大需求同周硬刚 |
| 大版本 / 破坏性变更 | 单独评估：差异清单 `high` 项逐项验证 |

团队统一使用 **merge** 将 `upstream/main` 合入同步分支（不对 `main` 做长期 rebase，避免共享历史被改写）。

### 4.2 标准步骤

```bash
# 1. 更新远程引用
git fetch origin
git fetch upstream

# 2. 基于最新 main 开同步分支
git checkout main
git pull origin main
git checkout -b sync/upstream-$(date +%Y%m%d)

# 3. 合并上游（不要在此提交业务功能）
git merge upstream/main
```

若有冲突：

1. 打开 [差异清单](./diff-inventory.md)，将可能波及的 `active` 项标为 `needs-rebase`。
2. 按文件解决冲突：
   - **上游安全 / 计费 / 协议兼容修复** → 默认采上游，再把本仓定制重放回去
   - **本仓独立文件** → 保留本仓
   - **两边都改同一逻辑** → 先理解上游意图，再最小 diff 合并，禁止「整文件选一边」糊弄
3. 编译与测试（见下节）。
4. 更新差异清单元信息（基准 commit/版本）及各项状态。
5. 开 PR → 合入 `origin/main`（同步 PR 标题建议：`sync: merge upstream/main @ <short-sha>`）。

```bash
# 4. 推送并开 PR（仅 origin）
git push -u origin HEAD
```

### 4.3 同步分支禁止事项

- 禁止夹带 `feat/` / 业务重构
- 禁止 `git push upstream`
- 禁止 `reset --hard` 丢弃未备份的本仓提交
- 禁止为「省事」删除差异清单中的 `active` 项而不改代码

### 4.4 同步后最小验证

在合并同步 PR 前至少完成：

```bash
# 与冲突相关的包测试
go test ./common/...
go test ./service/...
go test ./model/...
go test ./relay/...

# 若动到前端
cd web/default && bun run typecheck
```

手工烟雾（环境允许时）：

1. 登录 / API Token 鉴权
2. 一条 chat（或主推模型）成功且扣费/日志正常
3. 管理后台关键列表可打开
4. 差异清单中全部 `active` 项的「回归要点」

---

## 5. 发版与部署注意

1. **备份数据库**后再升级二进制/镜像。
2. 确认迁移在 SQLite / MySQL / PostgreSQL 目标环境可接受（本项目需三库思维，即使生产只用其一）。
3. 版本号建议带本仓后缀，例如 `x.y.z-ggapi.N`，并与清单「基准 upstream 版本」可对照。
4. 发版回归最小集：
   - 登录与 Token
   - 主链路推理 + 计费
   - 充值/额度变更（若有相关定制）
   - 清单 `active` 项
5. 观察错误日志与额度异常（含 `quota_saturation` 类审计，若启用）。

构建参考：

```bash
make build-web          # 默认前端
# make build-all-web    # 含 classic
# 镜像构建按仓库 Dockerfile / compose 执行
```

---

## 6. 向官方贡献

通用修复或可上游化的功能，建议回馈官方以降低长期分叉：

1. 先搜索上游 Issues / PRs，避免重复。
2. 大功能先开 Issue 对齐；Bug 关联 Issue。
3. 使用 `.github/PULL_REQUEST_TEMPLATE.md`，人工撰写摘要与验证证明。
4. 安全漏洞：**不要**公开 Issue，按 [`.github/SECURITY.md`](../../.github/SECURITY.md) 走 Advisory 或邮件。
5. 官方合入后，在本仓删除对应补丁，清单状态改为 `upstreamed`。

---

## 7. 命令速查

| 场景 | 命令 |
|------|------|
| 开发 API | `make dev-api` |
| 开发默认前端 | `make dev-web` |
| 重建 API 容器 | `make dev-api-rebuild` |
| 构建前端 | `make build-web` |
| 更新远程引用 | `git fetch origin && git fetch upstream` |
| 看上游多出的提交 | 先 `git fetch upstream`，再 `git log --oneline main..upstream/main` |
| 看本仓多出的提交 | 先 `git fetch upstream`，再 `git log --oneline upstream/main..main` |

---

## 8. 相关文档

- [差异清单模板](./diff-inventory.md)
- [二开文档索引](./README.md)
- [`AGENTS.md`](../../AGENTS.md)
- [Agent skill：ggapi-fork](../../.agents/skills/ggapi-fork/SKILL.md)（`/ggapi-fork` 逐步教练）
- [PR 模板](../../.github/PULL_REQUEST_TEMPLATE.md)
- [安全披露](../../.github/SECURITY.md)
