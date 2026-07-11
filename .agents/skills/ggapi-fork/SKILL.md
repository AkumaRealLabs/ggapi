---
name: ggapi-fork
description: >
  End-to-end playbook for maintaining the ggapi fork of QuantumNous/new-api:
  daily feature/fix workflow, branch naming, push-vs-PR decisions, pre-commit
  Codex review loop (/codex:review → fix → re-review), upstream sync/merge,
  conflict resolution, diff-inventory updates, release regression, contributing
  back upstream, org-linux CI runners, private-repo update-check PAT, common
  git/remote/billing/hotspot pitfalls, and controlled self-upgrade of this
  skill (Mode I) without chaotic rewrites.
  Use when the user runs /ggapi-fork, asks how to 二开, fork 开发, 开分支,
  提 PR, push 还是 PR, 提交, commit, 最终提交, 提交前审查, codex review,
  同步上游, merge upstream, 冲突解决, 差异清单, diff-inventory, 发版回归,
  origin/upstream 远程, org-linux, runner group, 检查更新, GitHub PAT, 升级
  skill, 改进 ggapi-fork, skill 自升级, 自检 skill, 或 any long-term fork
  maintenance question on this repository. Always load this skill before
  advising or executing fork workflow steps — including plain “提交/commit”
  on this repo (Mode C C2-pre). When improving this skill itself, follow
  Mode I / self-upgrade policy.
metadata:
  skill_version: "1.3.0"
---

# ggapi Fork Maintenance Playbook

This skill is the **agent operating manual** for long-term ggapi 二开.
Human-facing canonical docs live under `docs/fork/`; **do not invent process**
that contradicts them. Prefer acting as a step-by-step coach: state the current
mode, the next concrete command, and the stop/confirm points.

**Self-upgrade:** This skill may improve its own files under a **controlled**
policy (`references/self-upgrade.md`, Mode **I**). It must **not** randomly
rewrite rules, weaken safety, auto-commit, or invent process not in `docs/fork/`.

## Mandatory preflight (every invocation)

1. Confirm repo is **ggapi** (`origin` → `AkumaRealLabs/ggapi`, ideally
   `upstream` → `QuantumNous/new-api`).
2. Read (or re-read if stale in context):
   - `docs/fork/README.md`
   - `docs/fork/branch-and-sync-sop.md`
   - `docs/fork/diff-inventory.md` (meta + formal table)
   - `AGENTS.md` section **ggapi 二开维护** + Project Governance
3. Load the matching reference file(s) under this skill's `references/` for the
   active mode (see Mode router).
4. Run a short status snapshot before advising irreversible steps:

```bash
git remote -v
git branch --show-current
git status -sb
git fetch origin 2>/dev/null; git fetch upstream 2>/dev/null
git rev-parse --short HEAD
git rev-parse --short upstream/main 2>/dev/null || true
```

5. If `upstream` is missing, **stop coding features** and complete remote setup
   first (Mode A).

## Hard rules (never violate)

| Rule | Detail |
|------|--------|
| Push target | **Only** `origin`. Never `git push upstream`. |
| `main` protection | **Never** commit/push directly to `main` for normal work. Branch → PR → merge. |
| Thin customization | Prefer config / independent packages / new channel dirs over patching hotspots. |
| Hotspots | Extra care: `service/*quota*`, `model/` locks/transactions, `relay/`, auth middleware, `pkg/billingexpr/`. Billing → read `pkg/billingexpr/expr.md` first. |
| Diff inventory | Permanent fork diffs must be registered in `docs/fork/diff-inventory.md`. Sync updates baseline commit/date. |
| Branding | Never remove/replace **new-api** / **QuantumNous** protected identity (AGENTS.md). |
| Engineering rules | JSON via `common/*`, three DBs, billing safety, frontend i18n — all still apply. |
| Confirm before risk | Force-push, `reset --hard`, deploy, production DB, or anything shared: **ask user first**. |
| Commit/PR policy | Only commit/push/open PR when the user explicitly asks. Follow repo commit/PR rules. |
| Skill self-upgrade | May improve `.agents/skills/ggapi-fork/**` only per `references/self-upgrade.md`. **Never** silent L2/L3 rewrites; **never** auto-commit/push; **never** weaken Hard rules to “make it easier”. Prefer aligning skill → docs, not docs → bad skill. |

### Ship quality gate (Mode C — not a Hard rule)

Before the **final** commit of a ship unit (user said 提交 / commit / 最终提交 —
even without `/ggapi-fork`), run the **C2-pre Codex gate**: companion CLI
(preferred) or user `/codex:review`, review the **combined** final tree vs
`origin/main` (see workflows), fix findings, re-review until clean or escalate.
Docs/skill not auto-exempt. Does **not** auto-commit or replace Hard rules.
Full procedure: Mode C **C2-pre**.

## Mode router

Detect intent from the user message. If ambiguous, ask **one** clarifying
question with the mode list. Then open the listed reference and execute that
workflow end-to-end.

| Mode | User signals | Load |
|------|--------------|------|
| **A. Bootstrap** | 首次配置, remote, 环境, 怎么开始二开 | `references/workflows.md` §A |
| **B. Daily feature/fix** | 新功能, bug, 开分支, 实现, 本地验证 | `references/workflows.md` §B |
| **C. Ship (commit + push + PR)** | 提交, commit, 最终提交, push, PR, 合入, push 还是 PR | `references/workflows.md` §C + `references/checklists.md` §Pre-PR |
| **D. Upstream sync** | 同步上游, merge upstream, sync | `references/workflows.md` §D + `references/checklists.md` §Post-sync |
| **E. Conflict / inventory** | 冲突, 差异清单, GG-xxx, needs-rebase | `references/workflows.md` §E + `references/troubleshooting.md` |
| **F. Release** | 发版, 部署, 回归 | `references/workflows.md` §F + `references/checklists.md` §Release |
| **G. Contribute upstream** | 回馈官方, 向上游 PR | `references/workflows.md` §G |
| **H. Stuck / diagnose** | 报错, 推不了, 分叉乱了, 不知道下一步 | `references/troubleshooting.md` first, then re-route |
| **I. Skill self-upgrade** | 升级 skill, 改进 ggapi-fork, 自升级, 自检 skill, skill 与文档不一致 | `references/self-upgrade.md` + `references/CHANGELOG.md` |

If the user only asks a conceptual question ("到底应该 PR 还是 push？"), answer
from Mode C hard decision table **without** running push/PR unless they ask.

### Mode I quick gate (self-upgrade)

1. Load `references/self-upgrade.md` fully before editing this skill.  
2. Classify change **L0 / L1 / L2 / L3**.  
3. **L0:** explain gap only. **L1:** may edit skill files after one-line notice; no commit. **L2/L3:** written plan → user confirms → then edit.  
4. Always bump `metadata.skill_version` + `references/CHANGELOG.md` when files change.  
5. Ship skill changes via `docs/…` branch + PR like any other docs change; do not mix with unrelated features unless user insists.  
6. If `docs/fork` must change for truth: edit docs first (or same PR), skill second.

## Coaching language (user-facing)

- **Default for this repo’s human maintainer:** coach in **简体中文**.
- Match the user’s latest message language if they clearly switch (e.g. all-English
  question → English reply). Mixed CN/EN from the user → still prefer 中文.
- Keep **commands, branch names, paths, git output, and inventory field values**
  in their literal form (do not translate `feat/`, `origin`, `GG-001`, etc.).
- Internal skill/reference files may stay English; **what you say to the user**
  must be clear 中文 steps unless they asked otherwise.
- Progress frames, decision tables, and “下一步” prompts → 中文.

## Coaching style

For multi-step work, present progress as:

```text
当前模式: <B/C/D/…>
已完成: …
下一步: <具体命令或决策>
需要你确认: <仅风险操作 / 产品选择>
停止条件: <何时暂停>
```

Do **not** dump the entire SOP unprompted. Walk one phase at a time unless the
user asks for the full map.

## Decision: push vs PR (quick answer)

```text
功能/文档/同步工作在分支上完成
        │
        ▼
  git push -u origin HEAD     ← 推的是「分支」，不是 main
        │
        ▼
  开 PR → base: main          ← 唯一合入 main 的正规路径
        │
        ▼
  Review / CI → merge → 删分支 → 必要时更新 diff-inventory
```

| Action | Allowed? |
|--------|----------|
| `git push -u origin feat/…` (or current topic branch) | Yes |
| Open PR into `main` | Yes (required for normal land) |
| `git push origin main` / commit on `main` for routine work | **No** |
| `git push upstream …` | **Never** |

Full ship checklist: `references/checklists.md` §Pre-PR.

## Where customization should land

Prefer this order when implementing 二开 features:

1. **Config / env / setting** — no code fork if possible  
2. **New independent path** — e.g. `relay/channel/<new>/`, `pkg/<own>/`, `oauth/<new>/`, frontend feature module  
3. **Thin registration hook** — one-line router/registry touch + isolated impl (`medium` inventory risk)  
4. **Patch upstream file** — last resort; mark `patch` + risk in inventory; plan for rebase pain  

Never "drive-by" edit billing/auth/relay core without inventory + dual review plan.

## Fork inventory anchors (read, do not invent)

Permanent fork deltas live in `docs/fork/diff-inventory.md`. Skill must stay
aligned with active rows (not replace them):

| ID | Topic |
|----|--------|
| GG-001 | `docs/fork/` + `AGENTS.md` 二开入口 |
| GG-002 | this skill (`/ggapi-fork`, v1.3.0+ pre-commit Codex gate) |
| GG-003 | CI `runs-on.group: org-linux`, Linux amd64 only |
| GG-004 | Server-side update check URL + GitHub PAT |

## Related skills (hand off, do not reimplement)

| Topic | Skill |
|-------|--------|
| Pre-commit / final-ship code review (Codex) | **`/codex:review`** (Codex plugin; review-only — this skill owns the fix→re-review loop) |
| Adversarial / custom-focus Codex review | `/codex:adversarial-review` (optional; not the default gate) |
| Bundled local/PR reviewer (non-Codex) | `review` / `/review` — optional extra; does **not** replace `/codex:review` for the pre-commit gate |
| Frontend i18n keys (all locales incl. **zh-TW**) | `i18n-translate` |
| classic → default UI port | `classic-to-default-sync` |
| shadcn/ui in `web/default` | `shadcn-ui` |
| React performance | `vercel-react-best-practices` |
| Verify finished work | `check-work` / `/check-work` |

## Canonical docs (source of truth)

| Doc | Role |
|-----|------|
| `docs/fork/README.md` | Fork doc index + principles |
| `docs/fork/branch-and-sync-sop.md` | Branch names, daily loop, sync, release |
| `docs/fork/diff-inventory.md` | Permanent diffs + baseline upstream SHA |
| `AGENTS.md` | Engineering rules + protected branding |
| `pkg/billingexpr/expr.md` | Billing expression system (read before quota work) |
| `.github/PULL_REQUEST_TEMPLATE.md` | Official-style PR structure when contributing upstream |
| `.github/SECURITY.md` | Vulnerability disclosure |

If skill text and `docs/fork/*` disagree, **docs win** — then offer Mode **I**
(L1 align skill to docs). Do not “fix” docs to match a wrong skill unless the
user explicitly wants an SOP change.

## After completing a mode

Always leave the user with:

1. Current branch + whether it is pushed  
2. Whether `diff-inventory` needs an update  
3. Exact next command **or** "done for this mode"  
4. If they finished a permanent customization: remind GG-xxx registration before merge to `main`  
5. If a skill gap was found: optional one-line Mode I offer (do not force upgrade)
