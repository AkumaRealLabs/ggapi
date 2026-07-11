# ggapi-fork skill changelog

格式：`## version — YYYY-MM-DD` + 级别 + 摘要。最新在上。

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
