# ggapi-fork skill changelog

格式：`## version — YYYY-MM-DD` + 级别 + 摘要。最新在上。

## 1.5.0 — 2026-07-12

- **级别:** L2 / minor（Mode F 发版 tag 流程 + SOP §5.1；**未**改 Hard rules）  
- **原因:** 用户确认本仓 git tag 采用 `v<上游基线>.N`（例 `v1.0.0-rc.20.1`），不再推荐 `x.y.z-ggapi.N`  
- **变更:**
  - `docs/fork/branch-and-sync-sop.md` 新增 **§5.1**（格式、禁撞名、`N` 递增/仅基线版本串变化时归 1、ancestry 校验、**正式 tag 钉 `origin/main` tip**、打 tag 命令；Docker 与二进制发版路径分离）  
  - `docs/fork/diff-inventory.md`：当前基线记为 **`v1.0.0-rc.20`**（`ad900bbb`；`rc.21` 尚非 main 祖先）+「本仓版本策略」；GG-003 记 docker-build 防护  
  - `.github/workflows/docker-build.yml` + `docker-image-branch.yml`：**关闭 tag 自动触发**（前者）；两文件在 image 仍为上游 `calciumion/new-api` 时 **fail-closed**（避免 fork 推上游 Hub / 覆盖 `:latest`）  
  - Mode F / Release checklist 与 SOP 对齐；`N` **不**因每次 sync 无条件归 1  
  - `skill_version` → **1.5.0**  
- **未改:** Hard rules；不自动 commit/push；不自动打 tag  

## 1.4.0 — 2026-07-12

- **级别:** L2 / minor（第三壳教练 + #7 发车实战；**未**改 Hard rules 表与 origin/main/upstream/品牌/计费语义）  
- **原因:** 用户确认 Mode I 方案；`docs/fork` 已有 GG-005 / SOP §3.5，skill 仍教 `web/default`；吸收 PR #7 审查与合入教训  
- **变更:**
  - SKILL：`skill_version` → **1.4.0**；inventory 锚点 **GG-005**；Product shell 表；Mode I 触发词 **自提升**；description 含第三壳  
  - Mode A/B：默认本地栈与 typecheck → **`web/ggapi`**；功能追 default 的 port 提示  
  - Mode C：**C1b** 三壳/embed/品牌/i18n 检查（**置于 C2-pre 之前**；改后须再审）；materialize 临时 commit 禁止推送 temp 信息  
  - Mode D/E/F：sync 不夹带 ggapi 大皮肤；post-sync port follow-up；release **`make build-all-web`**（本地 go embed 三壳齐全）  
  - Mode B：typecheck 用 subshell，`make` 始终在仓库根；按**实际编辑的壳**选 `build-web*`  
  - checklists：第三壳 Pre-PR 段；双壳 theme 选择器；post-sync / release 勾选项  
  - troubleshooting **§22** 第三壳/主题/embed（default+ggapi 管理端均需 ggapi 选项）；§20 补 orphan keys + ggapi 路径  
  - `docs/fork/branch-and-sync-sop.md` 发版/命令速查对齐 `make build-all-web` 与产品壳 dev  
  - 依赖 skill：**i18n-translate** / **shadcn-ui** shell-aware（每条命令写全路径 `web/ggapi` 或 `web/default`）；i18n 含 **zh-TW**；classic 走 `i18next-cli`  
  - C1b / §22：classic `frontendTheme` 切换路径纳入 theme 门禁  
  - `web/{ggapi,default}/scripts/sync-i18n.mjs`：`zh-TW` 纳入 untranslated 非拉丁判定；字面量 allowlist 补 `Webhook`/`Gotify`  
  - GG-002 清单摘要对齐 v1.4.0  
- **未改:** Hard rules；不自动 commit/push/merge；C2-pre 仍可用户明确 opt-out  

## 1.3.0 — 2026-07-11

- **级别:** L2 / minor（发车流程 / Ship quality gate；**未**改 Hard rules 表与 origin/main/upstream/品牌/计费语义 → 非 L3；self-upgrade §2 + §6 已对齐「质量门禁≠安全语义 major」）  
- **原因:** 用户要求每次最终提交前跑 `/codex:review`，有问题则修复后再审  
- **变更:**
  - Mode C 新增 **C2-pre** + SKILL「Ship quality gate」（审查环，**不在** Hard rules 表内）  
  - 触发：`提交`/`commit` 进入 Mode C；agent 用 **companion CLI**（不依赖 slash）  
  - 范围：**单次** base→最终树审查；混合「已提交 + 脏工作区」时先 **materialize** 再 `--base origin/main`  
  - 文档/skill **不可**单方面跳过（须用户明确 opt-out）  
  - checklists / troubleshooting **§21**；`docs/fork` README+SOP；GG-002 → v1.3.0  
  - `skill_version` → 1.3.0  
- **未改:** Hard rules 表（origin-only / 禁直推 main / 品牌 / 计费 / 自升级红线）；不自动 commit/push；审查本身仍为 review-only  

## 1.2.0 — 2026-07-11

- **级别:** L2（含 L1 实战条目）  
- **原因:** 用户确认 Mode I 方案；吸收本会话已合入的 GG-003/GG-004 与发车踩坑  
- **变更:**
  - Mode C：`gh` 必须指向 `AkumaRealLabs/ggapi`；提供 `gh api repos/…/pulls` 回退；合并清理标准命令  
  - troubleshooting §17–§20：`gh` 错仓、org-linux CI、私有仓检查更新 PAT、i18n zh-TW  
  - checklists：目标仓 / zh-TW / GG-003 / GG-004 勾选项  
  - SKILL：触发词与 inventory 锚点 GG-001–004；`skill_version` → 1.2.0  
- **未改:** Hard rules、中文教练默认、不自动 commit/push  

## 1.1.0 — 2026-07-11

- **级别:** L2  
- **原因:** 用户要求 skill「能自主升级、但不乱升级」  
- **变更:**
  - 新增 Mode **I**（skill 自升级）与触发词  
  - 新增 `references/self-upgrade.md`（分级 L0–L3、闸门、禁止项、与 docs 优先级）  
  - `SKILL.md` 增加自升级硬规则与 `metadata.skill_version`  
  - 本 CHANGELOG 启用  
- **未改:** origin/main/upstream 硬规则、中文教练默认、commit/push 仍须用户明确授权  

## 1.0.0 — 2026-07-11

- **级别:** 初始  
- **变更:** 首版 playbook（Modes A–H、workflows / troubleshooting / checklists、中文教练）  
