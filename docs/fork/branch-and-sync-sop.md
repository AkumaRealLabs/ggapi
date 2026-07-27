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
Issue/任务 → 分支 → 实现 → 本地验证 → 用户在新终端用自选 Agent 审查（有问题则修后重新手动审查）→ commit → push 分支 → PR → Review → 合入 main → 更新差异清单（如有）
```

提交前人工终端 Agent 门禁由 skill **ggapi-fork** Mode C（C2-pre）协调：当前 Agent 到达门禁后停止操作，请用户在新终端使用任意终端 Agent 审查完整最终改动，并等待用户把结果发回；当前 Agent 不代跑审查工具。用户回传无问题后再 commit；有 findings 则修复后再次等待手动复审。详见 skill `references/workflows.md` 与 `docs/fork/README.md` Agent 技能小节。

### 3.1 实现约束（摘要）

完整规则以 `AGENTS.md` 为准，二开时特别注意：

- 分层：`router → controller → service → model`
- 定制优先落在**独立目录/包**或扩展点（新 channel、新 oauth、前端 feature）
- 少改热点：`service/*quota*`、`model/` 锁与事务、`relay/` 核心、鉴权 middleware
- 计费改动先读 `pkg/billingexpr/expr.md`，并走完整预扣 → 结算 → 退款链路自检
- 数据库改动必须 SQLite / MySQL / PostgreSQL 三端可接受
- 前端：产品壳为 `web/ggapi`（见 §3.5）；`web/default` 尽量保持与上游一致以便同步。Bun、i18n、`typecheck` / lint 在**正在改的壳**上跑通

### 3.2 本地验证（按改动范围）

```bash
# 后端：按包测试，勿整仓无脑跑除非有必要
go test ./service/...
go test ./model/...
go test ./relay/...

# 前端（二开产品壳）
cd web/ggapi && bun run typecheck
# 以及项目约定的 lint 脚本
# 仅改上游壳对照时：cd web/default && bun run typecheck

# 本地联调
make dev-api         # Docker 开发 API 栈
make dev-web-ggapi   # ggapi 第三壳（推荐）
# make dev-web       # 上游 default 壳
# make dev           # API + ggapi 壳
```

### 3.5 前端第三壳：`web/ggapi` 与功能追 `default`

**策略（长期）：**

| 壳 | 角色 |
|----|------|
| `web/default` | 跟踪上游 UI/功能，同步时优先合入；**不在此做本仓视觉大改** |
| `web/classic` | 上游经典 Semi 壳，按需保留 |
| `web/ggapi` | **本仓产品壳**（默认 `theme.frontend=ggapi`）；视觉与产品定制落在这里；**功能上追 `default`** |

**日常开发：**

1. 新页面/新交互/本仓皮肤 → 改 `web/ggapi`（`make dev-web-ggapi`）。
2. 上游只改了 `web/default` 的功能 → 在同步或独立 `chore/port-default-*` 分支里，把等价改动 **port 到 `web/ggapi`**（可参考 skill `classic-to-default-sync` 的 diff 审阅思路：对 commit 做路径映射 `web/default` → `web/ggapi`）。
3. 禁止只改 `web/default` 却期望生产默认壳生效；生产默认是 `ggapi`。
4. 共用依赖版本放在 `web/package.json` 的 `catalog` / workspaces；三壳各自 `package.json` 名称独立（`ggapi-web`）。
5. 构建：`make build-web-ggapi`；全量 `make build-all-web`；Docker 含 `builder-ggapi` 阶段。

**上游同步时前端注意：**

1. `web/default`、`web/classic` 冲突按上游意图解决，再评估是否需 port 到 `web/ggapi`。
2. `web/ggapi` 为**本仓独占树**，上游不会直接改；勿在 `sync/upstream-*` 里夹带大视觉重构。
3. 同步 PR 合并后，若 `web/default` 有用户可见功能 diff，开 follow-up：`chore/port-default-<topic>`，清单可备注关联 GG-005。
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
3. **版本号 / git tag** 按 §5.1；发版后更新清单「基准 upstream 版本」与本仓最近 tag（若登记）。
4. 发版回归最小集：
   - 登录与 Token
   - 主链路推理 + 计费
   - 充值/额度变更（若有相关定制）
   - 清单 `active` 项
5. 观察错误日志与额度异常（含 `quota_saturation` 类审计，若启用）。

### 5.1 版本号与 git tag 规则

**正式格式（本仓约定）：**

```text
v<上游基线>.N
```

| 段 | 含义 | 示例 |
|----|------|------|
| `v` + 上游基线 | 清单「基准 upstream 版本」：最近一次完整同步所对齐的 **upstream 版本串**（优先 upstream release tag 名去掉仅用于说明的装饰后，与官方 tag 主体一致） | 若清单记为 `v1.0.0-rc.20` → 基线串 `1.0.0-rc.20` |
| `.N` | 在**同一上游基线版本串**上，本仓第几次发版（从 `1` 起） | `.1` `.2` |

**完整 tag 示例（格式示意，非「当前 main 必打」）：** `v1.0.0-rc.20.1`、`v1.0.0-rc.20.2`、`v1.0.0-rc.21.1`（仅当清单基线已切到 `rc.21` 且 main 已包含该 tag）

| 规则 | 说明 |
|------|------|
| 禁止撞名 | **不要**打与 upstream 已有 tag **完全相同** 的名字（例如官方已有 `v1.0.0-rc.20` 时，本仓至少用 `v1.0.0-rc.20.1`） |
| 仅本仓改动再发 | 上游基线版本串不变，`N` +1 |
| `N` 何时归 1 | **仅当清单「基准 upstream 版本」相对上一发本仓 tag 发生变化时** 才把 `N` 重置为 `1`。多次 `sync/upstream-*` 若仍对齐同一上游 tag/版本串，则继续 `N+1`，**不要**重复打已有的 `.1` |
| 基线必须真实包含 | 发版 commit 必须 **祖先包含** 所选上游基线：用下方校验；禁止用「上游已有、但尚未合入本仓」的 tag 当基线做文案 |
| **钉在 origin/main** | 正式 release tag **只打在** `origin/main` 当前 tip 上：`git fetch origin` 后 `HEAD` 必须等于 `origin/main`（见下方校验）。禁止在本地未推送的 main、topic 分支、或与 `origin/main` 分叉的 tip 上打正式 tag |
| CI（默认：镜像 + Release 元数据） | 推 **origin** 的正式 fork tag（**`<上游 tag>.N`**，例 `v1.0.0-rc.20.1`；BASE 可从 origin 或 `QuantumNous/new-api` 校验且为发版 commit 祖先）→ **`ghcr.io/<owner>/<repo>:<tag>`**（仅此版本号 tag；**不再**打 `:<tag>-amd64`）+ **GitHub Release 元数据**（无强制二进制，供后台「检查更新」读 `releases/latest`）。发版 commit 须在 **`origin/main` 历史内**；cosign 后用**已签名 digest** 在 tip 仍匹配时更新 **`:latest`**（同样不打 `latest-amd64`）。手动 rebuild **不**动 `:latest`。GHA build cache 导出失败 **不**拖垮已成功的 push（`cache-to … ignore-error=true`）。分支镜像 `branch-*-<hash>`。**禁止** Docker Hub `calciumion/new-api`（见 GG-003） |
| CI（可选：裸二进制） | `release.yml` **不**在 tag push 时跑；需要 bare `go` 附件时 **workflow_dispatch 且必填 fork tag** |
| 只推 origin | 永远不要把本仓 tag push 到 `upstream` |
| 与 skill 版本无关 | `/ggapi-fork` 的 `skill_version`（如 1.5.0）**不是**业务二进制 / 镜像 tag |

**打 tag 前校验与示例（把 `BASE` / `N` 换成清单与递增结果）：**

```bash
git fetch origin
git checkout main && git pull --ff-only origin main
git fetch upstream
git fetch origin --tags

# 0) 正式 tag 必须钉在 origin/main tip（禁止仅本地 main / 分叉 tip）
MAIN_SHA=$(git rev-parse origin/main)
test "$(git rev-parse HEAD)" = "$MAIN_SHA" || {
  echo "ERROR: HEAD != origin/main; push/merge first, then re-fetch"
  exit 1
}

# 1) 从清单读「基准 upstream 版本」，例如记为 v1.0.0-rc.20 → BASE=v1.0.0-rc.20
#    或用：git describe --tags --abbrev=0 <清单基准 commit>
#    校验：git merge-base --is-ancestor <tag> <清单基准 commit>
BASE=v1.0.0-rc.20   # 示例占位：必须换成清单中真实、且已被 origin/main 包含的基线

# 2) 确认发版点（origin/main）包含该上游 tag（ancestry）
git merge-base --is-ancestor "$BASE" HEAD || {
  echo "ERROR: $BASE is not an ancestor of HEAD; update inventory baseline or sync first"
  exit 1
}

# 3) 选 N：同 BASE 已有本仓 tag 则取最大 N+1，否则 N=1
#    本仓 tag 形如 ${BASE}.N （例 v1.0.0-rc.20.1）
N=1   # 按 origin 已有 tags 递增后填写
REL_TAG="${BASE}.${N}"

# 4) 打本地 annotated tag（失败必须中止：同名 tag 已存在时勿继续 push）
git tag -a "$REL_TAG" -m "release based on upstream ${BASE}" "$MAIN_SHA" || {
  echo "ERROR: cannot create $REL_TAG (already exists locally?). Fix N or delete wrong local tag"
  exit 1
}

# 5) 推 tag 前用 ls-remote 再确认 tip（缩小 TOCTOU；仍非服务端严格 CAS）
#    勿 push ${MAIN_SHA}:refs/heads/main —— 在 main 被 force/删建时可能改写远端 main
REMOTE_MAIN=$(git ls-remote origin refs/heads/main | awk '{print $1}')
test -n "$REMOTE_MAIN" && test "$REMOTE_MAIN" = "$MAIN_SHA" || {
  git tag -d "$REL_TAG"
  echo "ERROR: origin/main is '${REMOTE_MAIN:-missing}', expected $MAIN_SHA; restart from step 0"
  exit 1
}
git push origin "refs/tags/${REL_TAG}:refs/tags/${REL_TAG}" || {
  git tag -d "$REL_TAG" 2>/dev/null || true
  echo "ERROR: tag push failed (remote tag exists?); restart after git fetch --tags"
  exit 1
}
# 6) 推后抽检：main 若已前进，tag 仍钉在 MAIN_SHA（当时 tip）；由人决定是否删 tag 重发
POST_MAIN=$(git ls-remote origin refs/heads/main | awk '{print $1}')
if [ "$POST_MAIN" != "$MAIN_SHA" ]; then
  echo "WARNING: origin/main moved to $POST_MAIN after tag push; $REL_TAG still points at $MAIN_SHA"
fi

# 7) 默认产物：GHCR 镜像（docker-build.yml）。拉取示例：
#    docker pull ghcr.io/akumareallabs/ggapi:${REL_TAG}
# 可选裸二进制：Actions → Release (Linux amd64 binaries) → Run workflow → 填 tag
```

历史曾讨论过的 `x.y.z-ggapi.N` **不再作为推荐格式**；新 tag 一律用本节 `v<上游基线>.N`。

构建参考（本仓 `main.go` embed **default + classic + ggapi** 三壳；`dist` 被 gitignore；**镜像 Dockerfile 已含三壳 builder**）：

```bash
# 默认发版：推 tag → CI 构建 GHCR 镜像（Dockerfile 内 build 三壳 + go）
# 本地验证镜像：
# docker build -t ggapi:local .
# 本地裸二进制（非默认 CI 路径）：
make build-all-web
# make build-web-ggapi  # 仅验证产品壳时
# make build-web        # 仅验证 upstream default 壳时
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
| 开发产品壳（推荐） | `make dev-web-ggapi` 或 `make dev` |
| 开发 upstream default 壳 | `make dev-web` |
| 重建 API 容器 | `make dev-api-rebuild` |
| 构建全部前端（发版/embed） | `make build-all-web` |
| 仅构建产品壳 | `make build-web-ggapi` |
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
