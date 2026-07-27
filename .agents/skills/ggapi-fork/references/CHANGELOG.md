# ggapi-fork skill changelog

格式：`## version — YYYY-MM-DD` + 级别 + 摘要。最新在上。

## 1.9.0 — 2026-07-27

- **级别:** L2 / minor（前端维护模型收敛；**未**改 Hard rules）
- **原因:** 用户决定不再维护第三前端壳，改为直接在上游 `v1.0.0-rc.22` 官方前端上保留最小 GG-004/006/007 功能差异
- **变更:**
  - 日常开发、验证、冲突解决与发版命令统一到官方 `web/`
  - GG-005 标记为 `dropped`；禁止恢复 `web/ggapi`、classic、多主题接线与独立视觉重构
  - C1b、Pre-PR、Post-sync、release 与 troubleshooting §20/§22 改为单前端检查
  - inventory 锚点补充 GG-006/GG-007，并更新 `docs/fork` 路径与 `v1.0.0-rc.22` 基线
  - `skill_version` → **1.9.0**
- **未改:** origin-only、禁直推 main、品牌/计费等 Hard rules；不自动 commit/push/merge；C2-pre 仍由用户在新终端手动审查并回传结果

## 1.8.0 — 2026-07-27

- **级别:** L2 / minor（Mode C 审查执行方替换；**未**改 Hard rules）
- **原因:** 用户要求当前 Agent 不再运行 Codex 或任何审查工具；到提交前审查节点后，改为请用户在新终端使用自选终端 Agent，并由用户把结果发回当前会话
- **变更:**
  - **C2-pre** 改为用户手动终端 Agent 审查交接：当前 Agent 暂存并指纹化完整最终树、给出直接提示后停止，未收到用户结果前不 commit
  - 用户回传 clean 才继续；回传 findings 则当前 Agent 修复、重测并再次停下等待用户手动复审；当前 Agent 不调用 companion、`codex review`、slash 或审查子 Agent
  - C0/C-continue 记录人工审查暂停前尚未执行的授权链；用户回传结果后按原链续跑，但结果本身不补出缺失的 push/PR/merge/发版授权
  - 保留 tree fingerprint、stale result 与 merge pin 防护；落后分支拒绝更新时按 merge base 审查，回传结果前重新 fetch 并校验独立记录的 main tip，交接时禁止遗留未暂存/未跟踪内容；C2 后改用 `HEAD^{tree}` + clean workspace 校验，不把正常的 staged → clean 误判为 stale
  - 取消为审查创建临时 commit 的要求
  - GG-003 当前态同步为所有现有 workflow 使用 GitHub 托管 `ubuntu-latest` Linux amd64；anti-slop 的标签/评论/关闭操作获得 `pull-requests: write`；不恢复已回退的全量 build/test CI 门禁
  - `docs/fork`、checklists、troubleshooting §21、self-upgrade 口径与 GG-002 同步
  - `skill_version` → **1.8.0**
- **未改:** origin-only、禁直推 main、品牌/计费等 Hard rules；不自动 commit/push/merge；审查仍可由用户明确 opt-out

## 1.7.0 — 2026-07-17

- **级别:** L2 / minor（Mode C 续跑 + C2-pre 强化；**未**改 Hard rules）
- **原因:** PR #26 全链路：`继续` 在 PR 已开时被当成再写代码；用户拒绝 agent 侧跳过 Codex；gate 须按树内容失效而非仅 SHA
- **变更:**
  - Mode **C-continue**：按阶段路由 `继续`（不默认 Mode B；不补 push/PR 词；C0 已含 merge 则续 C5；**仅**本单元跑过 C2-pre 时做 stale 检查；push-only 链不发明 gate）
  - **C2-pre**：禁 agent 自跳；修完再审；fingerprint=`write-tree`（含 untracked）；C2/C3 后更新 **merge pin**；C5 比对 remote `headRefOid`；`origin/main` SHA 漂移才 stale（非 count>0）；companion→CLI
  - checklists / troubleshooting §21 对齐；GG-002 → **v1.7.0**；`docs/fork/README` skill 版本锚点
  - `skill_version` → **1.7.0**（L2 走 minor，不用 1.6.1 patch）
- **未改:** Hard rules；不自动 commit/push/merge；C2-pre 仍可由**用户**明确 opt-out

## 1.6.0 — 2026-07-16

- **级别:** L2 / minor（Mode C/F 发车实战补齐；**未**改 Hard rules）
- **原因:** PR #23 全链路：oxlint 偏晚、`gh pr merge` GraphQL EOF 误判失败、tag 后未等 GHCR、一句话「提交→…→发版」缺链式语义
- **变更:**
  - Mode **B3** / **C1b** / Pre-PR：typecheck（ggapi/default）+ **路径级** oxlint/eslint（不全树 `bun run lint` 当硬门禁；基线有债）
  - Mode **C0**：链式发车（缺词不补权；**发版≠merge**；push/PR 不推断 commit）
  - Mode **C5** + **§23**：查 state/autoMerge；EOF ≠ 失败；**MERGED 前不发版**
  - Mode **F** + **§24**：`gh run list --branch $REL_TAG` + `watch --exit-status` 再称发版完成
  - classic：无 typecheck；勿套用 ggapi 命令
  - `skill_version` → **1.6.0**；GG-002 摘要对齐
- **未改:** Hard rules；不自动 commit/push/merge；不自动打 tag

## 1.5.1 — 2026-07-12

- **级别:** L1 / patch（Mode F 对齐 docs：默认 tag→GHCR；**未**改 Hard rules）
- **原因:** 用户要 tag 推 GHCR，且默认不构建裸二进制
- **变更:**
  - Mode F / Release checklist：默认产物 **GHCR**；`release.yml` 仅手动；禁 Docker Hub
  - 锚点 GG-003 文案；`skill_version` → **1.5.1**
- **未改:** Hard rules；不自动 commit/push

## 1.5.0 — 2026-07-12

- **级别:** L2 / minor（Mode F 发版 tag 流程 + SOP §5.1；**未**改 Hard rules）
- **原因:** 用户确认本仓 git tag 采用 `v<上游基线>.N`（例 `v1.0.0-rc.20.1`），不再推荐 `x.y.z-ggapi.N`
- **变更:**
  - `docs/fork/branch-and-sync-sop.md` 新增 **§5.1**
  - 清单基线 **`v1.0.0-rc.20`**；GG-003 Docker 防护（后续 1.5.1 / GHCR PR 再演进）
  - Mode F / Release checklist 与 SOP 对齐
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
