# ggapi-fork skill changelog

格式：`## version — YYYY-MM-DD` + 级别 + 摘要。最新在上。

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
