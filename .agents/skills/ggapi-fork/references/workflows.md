# ggapi Fork Workflows

Agent executes these sections when the matching mode is selected in `SKILL.md`.
Prefer real commands from the live repo (`makefile`, not assumed scripts).

**对用户输出默认简体中文**（见 `SKILL.md` Coaching language）：逐步说明、决策、清单勾选用中文；命令/路径/分支名保持原文。

---

## §A — Bootstrap (first-time / broken remotes)

### Goal

Local clone can develop against `origin` and safely fetch `upstream`.

### Steps

1. Verify remotes:

```bash
git remote -v
```

Expected:

| Remote | URL | Push |
|--------|-----|------|
| `origin` | `…/AkumaRealLabs/ggapi.git` | enabled |
| `upstream` | `…/QuantumNous/new-api.git` | **disabled** |

2. If `upstream` missing:

```bash
git remote add upstream https://github.com/QuantumNous/new-api.git
git remote set-url --push upstream DISABLED
git fetch upstream
```

3. If `origin` points at the wrong fork, **stop** and ask the user — do not
   rewrite remotes without confirmation.

4. Confirm branch hygiene:

```bash
git checkout main
git pull origin main
git status -sb
```

5. Optional local stack:

```bash
make dev-api         # API + docker dev stack
make dev-web-ggapi   # product shell (fork default; GG-005)
# make dev           # API + ggapi shell (makefile convenience)
# make dev-web       # upstream default shell only (sync/compare)
```

6. Point the user at `docs/fork/README.md` (含第三壳) and Mode B for first feature work.

### Done when

- `git fetch upstream` works
- push to `upstream` is impossible/disabled
- `main` tracks `origin/main`

---

## §B — Daily feature / fix

### Goal

Ship one coherent change on a short-lived branch without touching `main` directly.

### B0. Classify the change

Ask (or infer) and record:

| Question | Why |
|----------|-----|
| Feature / fix / docs / chore / hotfix? | Branch prefix |
| Permanent fork delta vs temporary experiment? | Inventory yes/no |
| Touches quota / auth / relay / model locks? | Dual review + `high` risk |
| Independent package possible? | Prefer thin customization |

### B1. Fresh branch from latest main

```bash
git fetch origin
git checkout main
git pull origin main
git checkout -b <prefix>/<short-topic>
```

Prefixes: `feat/` `fix/` `chore/` `docs/` `hotfix/`
Never start work on a dirty `main` with unrelated files — stash or finish first.

### B2. Implement with placement rules

1. Search for existing extension points before editing hotspots.
2. Prefer new files under own dirs.
3. Follow `AGENTS.md` (JSON wrappers, 3 DBs, pointer optional DTO fields, billing math helpers).
4. **Frontend shell (GG-005 / SOP §3.5):**
   - Product UI, skin, paper-sketch, operator-facing copy → **`web/ggapi`** (Bun, `t('English key')`, **i18n-translate**).
   - Keep `web/default` close to upstream for sync; do **not** land permanent product skin only in default and expect prod to show it (default theme is `ggapi`).
   - After upstream/`web/default` gains user-visible features: plan `chore/port-default-<topic>` into `web/ggapi` (path map `web/default` → `web/ggapi`; same review idea as **classic-to-default-sync**).
5. If permanent delta: draft inventory row mentally (ID, path, risk, regression).

### B3. Local verification (scoped)

Backend (only packages you touched):

```bash
go test ./service/...
go test ./model/...
go test ./relay/...
# or narrower: go test ./path/to/pkg -count=1
```

Frontend (if UI/TS touched — run on the shell you edited; keep make at **repo root**).
Align with `docs/fork` SOP: validation **per shell**; **do not** require a clean full-tree
`bun run lint` when the baseline already has pre-existing oxlint/format debt.

```bash
# typecheck (ggapi / default only — classic has no typecheck script)
(cd web/ggapi && bun run typecheck)     # product shell (usual)
# (cd web/default && bun run typecheck)

# Lint the **ship unit paths** (relative to the shell package), not necessarily the whole tree:
# ggapi/default — oxlint accepts file/dir args:
(cd web/ggapi && bunx oxlint -c .oxlintrc.json src/path/to/changed.tsx …)
# classic — prettier/eslint on changed paths if tooling allows; else note baseline:
# (cd web/classic && bunx eslint "src/…changed…" && bunx prettier --check "src/…")

# Optional full-tree lint (may fail on known baseline noise — not a ship blocker alone):
# (cd web/ggapi && bun run lint)

# Build the shell you actually edited (make always from repo root):
make build-web-ggapi   # if web/ggapi changed
# make build-web / make build-web-classic / make build-all-web as needed
```

**Ship lint rule:** every **new or modified** frontend file in the ship unit must be
clean under that shell’s linter (e.g. no nested ternary on touched ggapi files).
Unrelated pre-existing tree errors are **not** a hard stop — record them if you
ran full-tree and it failed. Fix findings on touched files before C2-pre.

Manual smoke when behavior is user-visible: login/token, one chat path, admin page (`make dev` or `make dev-web-ggapi` + API).

### B4. Diff-inventory draft (if permanent)

Before asking to commit/PR, prepare an entry for `docs/fork/diff-inventory.md`:

- New `GG-xxx` or update existing
- Risk `low|medium|high`
- Regression bullets

Do not invent secrets in the inventory.

### B5. Stop point

Present:

- Branch name
- File summary
- Test commands run
- Whether inventory update is included

**Do not commit** unless user asked. Then Mode C for ship.

---

## §C — Ship: commit → push branch → PR → merge hygiene

### Goal

Land work on `origin/main` via PR. Answer push-vs-PR without ambiguity.

### C0. Chained ship authorization (one user message)

When the user **in one message** stacks several Mode C/F verbs, treat that as
**one authorization chain** — run the named steps **in order** in the same
session without re-asking for each sub-step already listed.

| User phrase (examples) | Authorized steps |
|------------------------|------------------|
| 提交 / commit / 最终提交 | C2-pre → C2 only |
| 提交并 push / 提交并推上去 | C2-pre → C2 → C3 |
| 推上去 / push（**无**提交词） | **C3 only** on **existing** commits not on `origin/main`. If worktree dirty → **stop**, ask for 提交 (do **not** invent C2) |
| 开 PR / 提 PR（**无**提交词） | **C4** (push first if branch has unpushed commits). Dirty tree → **stop**, ask for 提交 |
| 开 PR 并合并 / 合并清理 | → through C5 (still no silent C2 if dirty without 提交) |
| 提交 → push → PR → merge | C2-pre → C2 → C3 → C4 → C5 |
| …→ merge → **发版** / 合并后打 tag | Same as merge chain **plus** Mode F only after PR is **MERGED** (merge verb **required**) |
| **发版** / tag / 打 tag / GHCR alone | Mode **F only** on already-landed `origin/main`. **Does not** authorize C5/merge |

Rules:

- **Missing verb = not authorized.** 「只提交」does **not** imply merge or 发版.
- **Push/PR wording ≠ commit.** 「推上去」「开 PR」alone never run C2-pre/C2 on a dirty tree; only move **already committed** work.
- **发版 ≠ merge.** 「提交并发版」without 合并/merge does **not** open C5; either the change is already on `origin/main`, or stop and ask for merge authorization.
- Still require a **topic branch** for routine ship (no routine commit/push to `main`).
- Still **never** `git push upstream`.
- Hard-risk ops (force-push, `reset --hard`, production deploy beyond tag/GHCR)
  still need their **own** explicit confirm even inside a chain.
- C2-pre still runs before any **new** final commit; docs/skill not auto-exempt.
- Mode F only when PR is **MERGED** and local `HEAD == origin/main` after fetch (never tag while PR still OPEN).

Coach one-liner after parsing:

```text
当前模式: C（链式发车）[→ F]
本句授权: 提交 / push / PR / merge / 发版（勾选实际出现的）
下一步: <first concrete command>
```

### C-continue. 「继续」when a ship unit is already in flight

User phrases: `继续` / `接着` / `/ggapi-fork 继续` **without** new feature/fix
scope.

**Detect stage first** (preflight: branch, dirty tree, open PR, CI).

**Resume rules:**

1. **C0 still wins** — `继续` never invents missing verbs (提交 / push / 开 PR / 合并 / 发版).
2. **If this unit ran C2-pre** (pass or recorded user skip): apply **Stale pass** before
   resumed C3/C4/C5. Stale → re-run gate (or stop); do not push/merge unreviewed content.
3. **If this unit never needed C2-pre** (e.g. C0 was push-only on already-committed clean
   tip, no new 提交): do **not** invent a gate on `继续`; resume authorized C3/C4/C5 only.
4. **C5 remote head:** after any push of the reviewed tree, record
   `git rev-parse HEAD` / remote tip as the **merge pin**. Before merge, fetch and require
   PR `headRefOid` == that pin (same rule as C5 table).

| Current stage | What 「继续」means | Do **not** |
|---------------|-------------------|------------|
| Mode B coding mid-feature, dirty or unfinished | Resume **B** (implement / verify) | Jump to commit without 提交 verb |
| Ready to ship, dirty tree, no commit auth yet | Coach next Mode C step; **stop** for 提交 if dirty | Invent C2-pre/C2 |
| Commits on branch, not pushed | **Coach only:** next is C3; push **only** if chain already has push auth **or** user says 推上去/push now. If unit had C2-pre, require non-stale. Prior **提交 alone** ≠ push | Silent C3; push stale gated unit |
| Branch pushed, no PR | **Coach only:** next is C4; open PR **only** if chain has 开 PR **or** user says 开 PR now. Gated units: non-stale | Silent C4 from 继续 alone |
| PR **OPEN**, gated unit still OK (clean pass **or** recorded skip for same tree; pin matches remote) | Report CI; if C0 already included 合并/merge → resume **C5**; else wait for **合并** | Invent merge; merge when pin ≠ remote head |
| PR OPEN, gated unit **stale** | Re-run C2-pre (or stop) before claim clean / push / merge | Claim “Codex 已通过” for unreviewed content |
| PR **MERGED**, local not cleaned | C5 hygiene if merge/cleanup authorized; else coach | Silent merge; 发版 without auth |
| On `main`, 发版 intent | Mode **F** only if 发版 authorized; else ask | |

**Lesson (PR #26):** after Codex clean + PR open, `/ggapi-fork 继续` = Mode C
post-ship (CI / merge auth / cleanup), **not** more product code unless the user
names a new defect.

### C1. Pre-flight

```bash
git branch --show-current   # must NOT be main (except emergency documented hotfix process)
git status -sb
git log --oneline origin/main..HEAD
git diff origin/main...HEAD --stat
```

If currently on `main` with commits: **stop**. Move commits to a topic branch:

```bash
git branch <prefix>/<topic>    # pointer to current commit
git reset --hard origin/main # ONLY if user confirms; destructive to main tip local
git checkout <prefix>/<topic>
```

Prefer safer alternative if unsure: ask user before any hard reset.

### C1b. Third-shell / embed ship checks (when `web/ggapi` or theme wiring touched)

Run **before C2-pre** for any GG-005-related unit (or theme/embed/Docker/makefile/release changes). If checks force code edits, finish those edits **then** run C2-pre so the gate covers the complete final tree (re-run C2-pre after any post-check fix).

| Check | Why (from #7 lessons) |
|-------|------------------------|
| `theme.frontend` accepts `ggapi` in backend **and** every shipped admin surface that can change theme: `web/ggapi` + `web/default` (schema / select / normalize) **and** classic helpers that write `theme.frontend` (e.g. `web/classic/src/helpers/frontendTheme.js` — today only switches to `default`; must not leave operators stuck off the product shell) | Admin on `default` or `classic` must still be able to reach product shell `ggapi` |
| Default theme remains intentional (`setting/.../theme.go` / constants) | Prod default is product shell |
| `makefile` + **`Dockerfile`** (+ optional `release.yml` if dispatching binaries) build **ggapi** dist before go embed | Missing Dockerfile ggapi stage → empty/missing embed in GHCR image |
| Protected branding: `index.html` title/meta and logo accessible name stay **New API** / QuantumNous policy | Fork skin ≠ stripping protected identity |
| New i18n keys live under locale `translation`; all shipped locales (incl. **zh-TW**) have key parity | Orphan top-level keys + missing locales caused review findings |
| Edited shell: **typecheck** on ggapi/default when TS touched; **lint ship-unit paths** (ggapi/default: `bunx oxlint -c .oxlintrc.json <paths>`; classic: eslint/prettier on paths — no `typecheck`) | Full-tree `bun run lint` is baseline-noisy; require clean **touched** files only |

Canonical policy: `docs/fork` §第三壳 / SOP §3.5; inventory **GG-005**.

### C2-pre. Final-commit Codex review gate (required before C2)

**When:** User asked to **commit / 提交 / 最终提交** (or “改完并提交…”, plain
`/commit`, “提交这些改动”) for a ship unit on this repo. **Mode C applies even
if the user did not say `/ggapi-fork`** — load this skill and run C2-pre before
any final commit. There must be reviewable work (dirty tree and/or commits not
on `origin/main`).

**Why:** Catch real defects before they enter history. Codex review is
**review-only** (does not patch). **This skill** owns the fix → re-review loop.

#### How to invoke (agent-callable — do not rely on slash alone)

The installed `/codex:review` slash command is often marked
`disable-model-invocation: true` (user-only). **Agents must not stall waiting
for the user to type the slash command.** Prefer the companion (same backend
as the slash command):

```bash
# Plugin root: installed Codex plugin path (e.g. ~/.grok/installed-plugins/codex-*)
export CLAUDE_PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$HOME/.grok/installed-plugins/codex-807cef0a}"
# Adjust codex-* dir if the install id differs; or discover:
# ls "$HOME/.grok/installed-plugins" | grep -E '^codex'

# Prefer foreground for the gate:
node "${CLAUDE_PLUGIN_ROOT}/scripts/codex-companion.mjs" review --wait
```

If companion path is wrong, fall back to `codex review` CLI with equivalent
scope flags per `codex review --help`. If **both** fail → troubleshooting §21
(not a silent pass).

User may still run `/codex:review --wait` themselves; treat that output as the
gate result for this round.

#### Scope: one combined target (base → final tree)

Plugin `auto` reviews **only** the working tree when dirty, and **only**
branch commits when clean. Two disjoint runs (worktree vs `HEAD` **plus**
`origin/main...HEAD`) **do not** equal one review of `origin/main` → final
worktree: interactions across commits + dirty fixes can be missed.

**Base freshness (before the gate counts as “vs current main”):**

```bash
git fetch origin
# commits on origin/main not in this branch:
git rev-list --count HEAD..origin/main
```

| `HEAD..origin/main` | Action |
|---------------------|--------|
| `0` | Proceed with scope table |
| `> 0` | **Prefer** merge `origin/main` into the topic branch (or rebase **only** if user allows), resolve, re-test, **then** review. If user declines update: still review, but coach must say gate is only vs **merge-base** of current tip, not a rebased-on-latest-main tree |

Do not claim “完整相对最新 main” unless the branch contains current `origin/main` (count was 0 after fetch, or merge/rebase done).

| Situation | What to run |
|-----------|-------------|
| Dirty only (no unique commits vs `origin/main`) | One review: working tree (default / `auto`) |
| Clean tree, commits on branch | One review: `review --wait --base origin/main` (or `--scope branch`) |
| **Both** dirty **and** commits not on `origin/main` | **Materialize** then **one** branch review (below) — do not stop at two disjoint reviews |
| Empty tree and nothing to land | Skip with reason |

**Materialize (mixed committed + dirty), only after user already authorized 提交:**

Goal: one Codex pass sees the full patch that will land (`origin/main` → final tree).

```bash
# 1) Stage only intentional ship files (never secrets)
git add <paths…>

# 2) Ephemeral commit so branch tip == intended final tree
git commit -m "chore: temp codex gate snapshot"

# 3) Single combined review
node "${CLAUDE_PLUGIN_ROOT}/scripts/codex-companion.mjs" review --wait --base origin/main
```

| Review result | Next |
|---------------|------|
| Findings | `git reset --soft HEAD~1` → fix in worktree → re-materialize from step 1 (counts as one cycle) |
| Clean | Keep tip; in **C2** amend to a proper why-focused message (`git commit --amend`) **only if** not pushed and hooks OK; or reset soft + one final commit with the real message |

Do not invent staged-only flags the plugin does not support. Do not leave the
temp message on a pushed branch. Temp message must not reach a pushed tip:
amend to a proper why-focused message in **C2**, or soft-reset and recommit,
**before** `git push`.

#### Loop

```text
用户授权「提交」（含纯 commit 话术，不要求先说 /ggapi-fork）
        │
        ▼
  有可审 diff？（dirty 和/或 origin/main...HEAD）
        │ 无 → 跳过并说明 → C2
        ▼
  用 companion（或用户 slash）跑 **一次** Codex review
  覆盖完整 ship unit（见上表；混合时先 materialize 再
  `--base origin/main`，禁止两段割裂审查当通过）
  优先 --wait
        │
        ├─ 无实质问题 / 仅 nit → 进入 C2
        │
        └─ 有实质问题
                │
                ▼
          修复（可再跑测试）
                │
                ▼
          再审（同样覆盖完整 ship unit）
                │
                └─ 仍有问题 → 再修再审
                   默认最多 3 轮；仍卡 → 停，展示报告，
                   不擅自 commit；用户可「带问题提交」或继续改
```

#### Rules

| Rule | Detail |
|------|--------|
| Trigger | Any final-commit intent on this repo → Mode C + C2-pre (not only “push/PR” wording). |
| Prefer wait | Small/medium diffs: `--wait` so the gate completes in-session. |
| Invocation | Agent uses **companion / `codex review` CLI**; slash is optional UX for humans. Companion auth/path fail → fall back to `codex review` CLI (same scope); both fail → §21, **not** pass. |
| Full unit | Always one combined base→final-tree review; mixed → materialize then `--base origin/main`. |
| Fix owner | **ggapi-fork agent** applies fixes; never claim review will edit code. |
| Re-review mandatory | After **any** fix batch that **changes ship-file content**, run Codex **again** on the full unit before C2 / push / “gate 通过”. |
| Stale pass | **Record at pass** (coach notes): content fingerprint of the **final ship tree** — stage **all intentional ship paths including untracked**, then `git write-tree` (preferred; covers dirty + new files). Also record `origin/main` SHA. **After C2/C3**, set **merge pin** = pushed tip SHA whose tree OID matches that fingerprint. Invalidate if ship content changes (fingerprint), if recorded `origin/main` SHA moves, or if before C5 remote `headRefOid` ≠ merge pin. Do **not** stale merely because `HEAD..origin/main > 0` when the user declined update against the same recorded main SHA. Message-only amend of the same tree does not stale. |
| Round limit | Default **3** full cycles (review → fix → re-review). Then stop and escalate. |
| No agent self-skip | Agent must **never** skip the gate, “带病通过”, or treat stall/timeout as pass. Only **user** clear opt-out (table below) or empty-tree case. |
| No silent skip | Skipping requires an allowed case below **and** a one-line reason. Docs/skill changes are **not** auto-exempt. |
| Still need auth | Passing the gate does **not** auto-commit; C2 still needs explicit commit intent. |
| Optional extra | `/codex:adversarial-review` or bundled `/review` only if user asks; not a substitute. |

#### Allowed skip (must state reason)

| Case | Skip? |
|------|--------|
| Empty tree and nothing to land | Yes — nothing to review |
| Codex plugin / CLI / companion unavailable or auth failure | Do **not** pretend passed: stop, §21; **default = 先不提交** until user chooses retry / install / 「跳过审查并提交」 |
| User explicitly:「跳过 Codex / 不审了直接提交 / skip review」 | Yes — record in reply; still do normal git hygiene. **Reject** agent-proposed skip if user instead says「不行得过 codex / 必须过审查」→ resume fix→re-review |

Gate is a **Mode C quality step** (user-confirmed L2 playbook), not a Hard-rule rewrite: user may always opt out with clear language; agent never auto-skip docs/skill.

**Not** an allowed unilateral skip: pure docs, skill-only, “small wording”,
“已经够好了”, time pressure, or companion flake — including changes to this gate.
Same gate unless the user opts out. Lesson (PR #26): user may **insist** the gate
re-run after a stalled or skipped attempt; treat that as mandatory re-entry.

#### Coach frame after gate

```text
当前模式: C（发车）
Codex 审查: 通过 / 已修 N 轮后通过 / 跳过（原因）/ 阻塞（见报告）
范围: working-tree | branch vs origin/main | materialized combined
下一步: commit（C2）或按报告继续改
```

If C1b (or any other check) still finds work after the gate and you edit files, **re-run C2-pre** before C2.

### C2. Commit (only if user asked)

**Only after C2-pre** (passed, allowed skip, or user override).

Follow repo commit rules:

- `git status` / `git diff` / `git log` style analysis
- Stage intentional files only
- HEREDOC commit message, why-focused
- Never update git config; never skip hooks

If change is permanent fork delta, include inventory file in the same PR when possible.

### C3. Push branch (not main)

```bash
git push -u origin HEAD
```

This is **required** before a PR can exist. Pushing a topic branch ≠ landing on main.

### C4. Open PR

**Critical:** `gh` may default to **upstream** `QuantumNous/new-api` in this
clone (module path / prior checkout). Always target **AkumaRealLabs/ggapi**.

```bash
# Prefer one-time fix for this clone:
gh repo set-default AkumaRealLabs/ggapi

# Verify before create:
gh repo view --json nameWithOwner -q .nameWithOwner
# must print: AkumaRealLabs/ggapi
```

If `gh pr create` fails with *No commits between main and docs/…* or blank SHAs
while `git log origin/main..HEAD` shows commits, **do not** open the PR against
upstream. Use the REST API against the team repo:

```bash
gh api repos/AkumaRealLabs/ggapi/pulls \
  -f title='…' \
  -f head="$(git branch --show-current)" \
  -f base='main' \
  -f body="$(cat <<'EOF'
## 摘要
- …

## 验证
- [ ] …

## 差异清单
- [ ] 无永久分叉 / 已更新 docs/fork/diff-inventory.md（GG-xxx）

## 风险
- 计费/鉴权/relay: 是/否；二审: 是/否
EOF
)"
```

Optional after set-default works:

```bash
gh pr create --repo AkumaRealLabs/ggapi --base main --head "$(git branch --show-current)" --title "<title>" --body "…"
```

Use `.github/PULL_REQUEST_TEMPLATE.md` structure when contributing **upstream**;
for internal ggapi PRs, keep the checklist above at minimum.

### C5. Merge + clean branches (only when user asks)

```bash
# Merge (example: PR number N)
gh pr merge <N> --repo AkumaRealLabs/ggapi --merge --delete-branch
```

**Do not treat CLI transport errors as merge failure.** `gh pr merge` can exit
non-zero with GraphQL EOF / timeout while GitHub still merged the PR (lesson:
PR #23). Always **verify** before retrying merge:

```bash
# Prefer gh pr view (GraphQL). If GraphQL stays down (same EOF class as merge):
gh pr view <N> --repo AkumaRealLabs/ggapi \
  --json state,mergedAt,mergeCommit,autoMergeRequest,mergeStateStatus,headRefOid \
  -q '{state:.state,mergedAt:.mergedAt,sha:.mergeCommit.oid,auto:.autoMergeRequest,mss:.mergeStateStatus,head:.headRefOid}'
# REST fallback (does not depend on GraphQL):
# gh api repos/AkumaRealLabs/ggapi/pulls/<N> \
#   -q '{state:.state,merged:.merged,sha:.merge_commit_sha,head:.head.sha}'
```

| `state` / signals | Action |
|-------------------|--------|
| `MERGED` / REST `merged: true` | Success → clean-up (do **not** re-merge) |
| `OPEN` + `autoMergeRequest` set / merge queue pending / `mergeStateStatus` waiting on checks | **Wait** (poll view or REST). Do **not** REST-force merge; do **not** enter Mode F or tag old `main` |
| `OPEN` + no auto-merge, CLI was transport flake | Retry `gh pr merge` once. REST only if user still wants immediate merge, and **pin the reviewed head**: `gh api -X PUT repos/AkumaRealLabs/ggapi/pulls/<N>/merge -f merge_method=merge -f sha='<mergePin>'` (or `gh pr merge --match-head-commit <oid>`). Before any merge: fetch and compare remote `headRefOid` to the **merge pin** (pushed tip whose tree matches the C2-pre fingerprint); if moved → **stop**, re-review; do not land unreviewed tip |
| `CLOSED` unmerged | Stop — not landed |
| Both GraphQL view and REST fail | Stop; do not guess MERGED; do not Mode F |

**C5 is not done until `state == MERGED`.** Chained 发版 must wait for MERGED
then `git pull origin main` before Mode F.

Then local hygiene:

```bash
git fetch origin
git checkout main
git pull origin main
git branch -d <topic-branch>              # local; ignore if already gone
git remote prune origin                   # drop origin/<topic> tracking
# if remote branch still listed:
# git push origin --delete <topic-branch>
```

Update inventory status if the PR introduced/changed permanent diffs and that
update was not already in the merged PR.

User phrases like「开 PR 并合并清理」mean: create PR → merge → delete remote/local
topic branch → leave `main` clean. Still require explicit merge wording; do not
merge on open alone. Full chains with 提交/push/发版: see **C0**.
If merge verify fails with flake symptoms → troubleshooting **§23**.
Ambiguous `继续` while PR is open: see **C-continue** (CI / wait for 合并 — not
silent merge, not re-code).

### Decision table (teach the user)

| Situation | Do this |
|-----------|---------|
| Finished coding on `feat/x` | `push` branch → open **PR** |
| Want code on `main` | Merge the **PR** (not direct push) |
| Docs-only on `docs/…` | Same: push branch + PR |
| Sync upstream branch | Same: push `sync/…` + PR |
| Emergency production break | `hotfix/…` branch still preferred; if direct `main` is unavoidable, user must explicitly authorize and document |

---

## §D — Upstream sync

### Goal

Merge `upstream/main` into ggapi without losing permanent customizations and
without pushing to upstream.

### D1. Cadence reminders

| Trigger | Action |
|---------|--------|
| Weekly | `git fetch upstream`; skim releases/commits |
| Security fix upstream | Open sync ASAP |
| Routine features | Biweekly/monthly window |
| Breaking major | Plan using inventory `high` rows first |

Use **merge**, not long-lived rebase of shared `main`.

### D2. Create sync branch

```bash
git fetch origin
git fetch upstream
git checkout main
git pull origin main
git checkout -b sync/upstream-$(date +%Y%m%d)
```

### D3. Merge upstream only (no feature work)

```bash
git merge upstream/main
```

Record range for PR body:

```bash
git rev-parse --short upstream/main
git log --oneline main..upstream/main | head -50
```

### D4. On conflicts → Mode E

Do not bulk "accept theirs/ours" on mixed files.

### D5. Validate

```bash
go test ./common/...
go test ./service/...
go test ./model/...
go test ./relay/...
# if frontend conflicted:
cd web/default && bun run typecheck
```

Smoke: login/token, one chat + billing log, admin channel list, every `active`
inventory regression note.

### D6. Update inventory meta

In `docs/fork/diff-inventory.md`:

- 最近更新日期
- 基准 upstream commit = merged upstream short SHA
- 基准 upstream 版本 if tag known
- Flip `needs-rebase` → `active` after verification
- Register any new permanent deltas introduced while resolving

### D7. Ship sync via PR

```bash
git push -u origin HEAD
# PR title: sync: merge upstream/main @ <short-sha>
```

**Forbidden on sync branch:** feature commits, push upstream, deleting `active`
inventory rows without code removal, `reset --hard` without backup/consent.

---

## §E — Conflicts & inventory discipline

### Conflict resolution order

1. Open `docs/fork/diff-inventory.md`; mark affected `active` rows `needs-rebase`.
2. Per conflicted file:

| File kind | Default resolution |
|-----------|--------------------|
| Upstream security / billing / protocol fix | Take upstream intent, **replay** fork patch on top |
| Fork-only path (`docs/fork/`, own `pkg/`, own channel) | Keep fork |
| Both changed same logic | Read upstream commit message; minimal merge; never whole-file blind pick |
| `AGENTS.md` | Keep ggapi 「二开维护」 section; merge upstream engineering rule edits |
| `web/default` / `web/classic` | Prefer upstream intent for shared shells; then assess **port** into `web/ggapi` (SOP §3.5) |
| `web/ggapi/**` | Fork-only product tree — upstream will not edit it; do **not** fold large skin refactors into the sync commit |

3. Build/test.
4. Restore inventory rows to `active` (or update summary).
5. Update baseline SHA/date.
6. If `web/default` gained user-visible features: open or queue follow-up `chore/port-default-<topic>` (GG-005); do not assume prod shell already has them.

### Inventory when-to-write

Register/update when **any**:

- Permanent code/config/docs vs upstream after merge to `main`
- Touch existing upstream files
- Add fork-only packages/channels/scripts
- Conflict resolution kept fork behavior

Statuses: `active` | `needs-rebase` | `upstreamed` | `dropped`
Types: `config` | `feature` | `patch` | `branding-safe` | `infra` | `docs`
Risk: `low` | `medium` | `high`

### ID rules

- Stable `GG-xxx`; never reuse closed IDs
- Template examples use `GG-xxx` / `GG-EXnn` only
- New real rows go in section **正式清单** only

---

## §F — Release / deploy notes

1. Backup DB before binary/image upgrade.
2. Migrations must be acceptable on SQLite / MySQL / PostgreSQL thinking even if prod uses one.
3. Version / git tag: **`v<upstream-baseline>.N`** (e.g. `v1.0.0-rc.20.1`) — see `docs/fork/branch-and-sync-sop.md` §5.1. Never reuse an exact upstream tag name. Bump `N` while the inventory baseline **version string** is unchanged; reset `N` to **1 only when that baseline version string changes**. Before tagging: `git fetch origin` and require `HEAD == origin/main`; ancestry check; `git tag -a` must succeed; `git ls-remote` tip re-check then push tag only; post-push tip warning if main moved (SOP §5.1).
4. **Default tag product = GHCR image** via `docker-build.yml` → `ghcr.io/<owner>/<repo>:<tag>` **plus metadata GitHub Release** (for update-checker `releases/latest`); `:latest` only when tip still matches after sign. Manual rebuild does not move `:latest`. Fork form `<upstream-tag>.N` only. **Never** Docker Hub `calciumion/new-api` (GG-003).
5. **Bare binary is optional:** `release.yml` is **workflow_dispatch + required tag** (attaches go binaries to the Release).
6. **After tag push — wait for GHCR before claiming 发版完成** (~8–15 min was the org-linux figure; hosted runners may differ; lesson: `v1.0.0-rc.21.7`):

```bash
REL_TAG=v1.0.0-rc.21.7   # the tag you just pushed
# Tag pushes set headBranch to the tag name — filter so you do not pick another run
gh run list --repo AkumaRealLabs/ggapi --workflow=docker-build.yml \
  --branch "$REL_TAG" --limit 5
# Take the run id for this tag / matching headSha, then:
gh run watch <run-id> --exit-status --repo AkumaRealLabs/ggapi
# --exit-status: non-zero if conclusion is failure/cancelled (do not ignore)
```

| Workflow conclusion | Coach action |
|---------------------|--------------|
| `success` (and `watch --exit-status` exit 0) | Report GHCR tag ready; optional `docker pull`; only then say 发版完成 |
| `failure` / `cancelled` (or watch exit ≠ 0) | Do **not** claim ship done; open logs (`gh run view <id> --log-failed`); Mode H / §24 |
| Still `in_progress` | Keep waiting or give user the run URL; do not invent success |
| No run for `$REL_TAG` | Do not use an unrelated recent success; diagnose §24 / §18 |

```bash
# Default: only after success — pull GHCR (package may be private — docker login ghcr.io)
# docker pull ghcr.io/akumareallabs/ggapi:v1.0.0-rc.21.7
```

7. Minimum regression:

- Login + API token
- Main inference path + billing
- Top-up/quota changes if customized
- All inventory `active` regression points
- Watch logs for quota saturation / auth errors

```bash
# Local bare binary (not default CI path): embeds need all three dist trees
make build-all-web
# Dockerfile already builds default + classic + ggapi (builder-ggapi) then go
```

Confirm Dockerfile still builds **ggapi** (and other embedded shells) on image path (GG-005 / #7).
Do not run production deploy commands without explicit user request and environment confirmation.
Chained「…→ merge → 发版」from Mode C: see **C0** — explicit merge verb required before C5; Mode F only on `origin/main` tip (never treat 发版 alone as merge auth).

---

## §G — Contribute back upstream

1. Search upstream issues/PRs first.
2. Prefer changes that reduce fork drift.
3. Branch from a clean cherry-pick or minimal patch against upstream if needed.
4. PR uses `.github/PULL_REQUEST_TEMPLATE.md`; human-written summary/tests.
5. Security: **no** public issue — follow `.github/SECURITY.md`.
6. After upstream merges: remove fork patch, set inventory status `upstreamed`, delete dead code in a ggapi PR.

Never push ggapi-only branding or private config to upstream.

---

## §I — Skill self-upgrade（可控）

完整策略见 `references/self-upgrade.md`。此处仅执行摘要：

1. **分级：** L0 只诊断 → L1 安全小补 → L2 扩展能力 → L3 动硬规则。
2. **可主动：** 文档漂移、实战证明 skill 错、用户要求升级/自检。
3. **不可乱来：** 业务 PR 顺手大改 skill、削弱 Hard rules、自动 commit/push、无文档依据发明流程。
4. **L1：** 一句话告知后可改文件 + CHANGELOG + bump version；**不**自动提交。
5. **L2/L3：** 先出方案，用户确认后再改；L3 必须独立说明规则变更。
6. **入库：** `docs/ggapi-fork-*` 分支 → 用户授权后 commit → push → PR → merge → 删分支。
7. **真相源：** `docs/fork/*` > skill > 会话临时话。
8. **对齐清单：** 升级后核对 GG-002 摘要与 `skill_version`；提及的永久差异（GG-003/004/005 等）须与 inventory 一致。
9. **触发词：** 「自提升」与「自升级 / 升级 skill / Mode I」同等进入本模式。
