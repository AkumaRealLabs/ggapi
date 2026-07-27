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
make dev-web         # official frontend
# make dev           # API + official frontend (makefile convenience)
```

6. Point the user at `docs/fork/README.md` (official single-frontend policy) and Mode B for first feature work.

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
4. **Official frontend (`web/`, SOP §3.5):**
   - Add GG-004/006/007 capabilities directly to the matching official feature modules (Bun, `t('English key')`, **i18n-translate**).
   - Prefer independent feature files and existing extension points; keep patches to upstream components minimal.
   - Do not recreate the dropped GG-005 third shell, classic frontend, runtime theme selector, or fork-only visual redesign.
5. If permanent delta: draft inventory row mentally (ID, path, risk, regression).

### B3. Local verification (scoped)

Backend (only packages you touched):

```bash
go test ./service/...
go test ./model/...
go test ./relay/...
# or narrower: go test ./path/to/pkg -count=1
```

Frontend (if UI/TS touched — run from `web/`; keep make at **repo root**).
Align with `docs/fork` SOP; **do not** require a clean full-tree
`bun run lint` when the baseline already has pre-existing oxlint/format debt.

```bash
# typecheck
(cd web && bun run typecheck)

# Lint the **ship unit paths** (relative to `web/`), not necessarily the whole tree:
(cd web && bunx oxlint -c .oxlintrc.json src/path/to/changed.tsx …)

# Optional full-tree lint (may fail on known baseline noise — not a ship blocker alone):
# (cd web && bun run lint)

# Build the official frontend (make always from repo root):
make build-web
```

**Ship lint rule:** every **new or modified** frontend file in the ship unit must be
clean under the frontend linter (e.g. no nested ternary on touched files).
Unrelated pre-existing tree errors are **not** a hard stop — record them if you
ran full-tree and it failed. Fix findings on touched files before C2-pre.

Manual smoke when behavior is user-visible: login/token, one chat path, admin page (`make dev` or `make dev-web` + API).

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
- C2-pre is an intentional cross-turn pause. Record the remaining authorized
  C0 chain before waiting. When the user returns the review result, resume that
  exact chain after stale checks; the result does **not** add missing
  commit/push/PR/merge/release verbs.
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
2. **If this unit ran C2-pre** (pass or recorded user skip): apply the matching
   **stale-result check** before resumed C3/C4/C5. Before C2, compare the index
   snapshot; after C2, compare `HEAD^{tree}` with the reviewed fingerprint and
   require a clean index/worktree. Stale → re-run gate (or stop); do not
   push/merge unreviewed content.
3. **If this unit never needed C2-pre** (e.g. C0 was push-only on already-committed clean
   tip, no new 提交): do **not** invent a gate on `继续`; resume authorized C3/C4/C5 only.
4. **If C2-pre is waiting for the user:** `继续` alone is not a result. A returned
   clean result or findings resumes the previously recorded C0 chain at the
   review-result handler; it never broadens that chain.
5. **C5 remote head:** after any push of the reviewed tree, record
   `git rev-parse HEAD` / remote tip as the **merge pin**. Before merge, fetch and require
   PR `headRefOid` == that pin (same rule as C5 table).

| Current stage | What 「继续」means | Do **not** |
|---------------|-------------------|------------|
| Mode B coding mid-feature, dirty or unfinished | Resume **B** (implement / verify) | Jump to commit without 提交 verb |
| Ready to ship, dirty tree, no commit auth yet | Coach next Mode C step; **stop** for 提交 if dirty | Invent C2-pre/C2 |
| C2-pre waiting for the user's terminal review result | Keep waiting; a returned clean result/findings resumes the recorded chain | Treat `继续` or silence as review pass |
| Commits on branch, not pushed | **Coach only:** next is C3; push **only** if chain already has push auth **or** user says 推上去/push now. If unit had C2-pre, require non-stale. Prior **提交 alone** ≠ push | Silent C3; push stale gated unit |
| Branch pushed, no PR | **Coach only:** next is C4; open PR **only** if chain has 开 PR **or** user says 开 PR now. Gated units: non-stale | Silent C4 from 继续 alone |
| PR **OPEN**, gated unit still OK (clean pass **or** recorded skip for same tree; pin matches remote) | Report CI; if C0 already included 合并/merge → resume **C5**; else wait for **合并** | Invent merge; merge when pin ≠ remote head |
| PR OPEN, gated unit **stale** | Re-run C2-pre (or stop) before claim clean / push / merge | Claim “审查已通过” for unreviewed content |
| PR **MERGED**, local not cleaned | C5 hygiene if merge/cleanup authorized; else coach | Silent merge; 发版 without auth |
| On `main`, 发版 intent | Mode **F** only if 发版 authorized; else ask | |

**Lesson (PR #26):** after the user reports a clean review + PR open, `/ggapi-fork 继续` = Mode C
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

### C1b. Frontend / embed ship checks (when `web/`, Docker, makefile, or release wiring is touched)

Run **before C2-pre** for frontend/embed/Docker/makefile/release changes. If checks force code edits, finish those edits **then** run C2-pre so the gate covers the complete final tree (re-run C2-pre after any post-check fix).

| Check | Why |
|-------|-----|
| `web/ggapi` and `web/classic` stay absent; `theme.frontend` remains only upstream's retired-option compatibility path | GG-005 must not be silently restored |
| `makefile` + **`Dockerfile`** (+ optional `release.yml` if dispatching binaries) build `web/dist` before Go embed | Missing official frontend dist breaks GHCR/bare binaries |
| Protected branding: `index.html` title/meta and logo accessible name stay **New API** / QuantumNous policy | Fork features do not remove protected identity |
| New i18n keys live under locale `translation`; all shipped locales (incl. **zh-TW**) have key parity | Orphan top-level keys + missing locales caused review findings |
| `web/`: **typecheck** when TS touched; **lint ship-unit paths** with `bunx oxlint -c .oxlintrc.json <paths>` | Full-tree `bun run lint` may be baseline-noisy; require clean **touched** files only |

Canonical policy: `docs/fork` SOP §3.5; inventory **GG-005** is dropped.

### C2-pre. User-run terminal-agent review gate (required before C2)

**When:** User asked to **commit / 提交 / 最终提交** (or “改完并提交…”, plain
`/commit`, “提交这些改动”) for a ship unit on this repo. **Mode C applies even
if the user did not say `/ggapi-fork`** — load this skill and stop at C2-pre
before any final commit. There must be reviewable work (dirty tree and/or
commits not on `origin/main`).

**Responsibility boundary:** The current Agent prepares the final review tree,
asks the user to open a new terminal and use any preferred terminal Agent, then
waits for the user to return the result. The current Agent **must not** invoke
Codex, a companion, another review command, or a review sub-agent. It owns only
the preparation, fixes, stale-result checks, and repeated handoff.

#### Prepare one combined target (base → final tree)

**Base freshness:**

```bash
git fetch origin
git rev-list --count HEAD..origin/main
git rev-parse origin/main
```

| `HEAD..origin/main` | Action |
|---------------------|--------|
| `0` | Use the current `origin/main` SHA as `REVIEW_BASE` |
| `> 0` | **Prefer** merge `origin/main` into the topic branch (or rebase only if the user allows), resolve, re-test, then restart preparation. If the user declines, set `REVIEW_BASE` to `git merge-base HEAD origin/main`; do not diff directly against latest `origin/main`, because main-only commits would appear as deletions |

Always record the fetched `origin/main` SHA separately as `REVIEW_MAIN_TIP`,
including when a declined update makes `REVIEW_BASE` the merge base. This tip is
the freshness anchor checked when the result returns.

After the user has authorized commit, stage **only** intentional ship paths,
including intended untracked files. Never stage secrets. The index is the final
tree handed to the user's reviewer, so one comparison covers existing branch
commits plus staged fixes:

```bash
git add <intentional-paths…>
git diff --cached --stat <REVIEW_BASE>
git diff --cached --check <REVIEW_BASE>
git diff --quiet                            # must exit 0: no unstaged tracked changes
git ls-files --others --exclude-standard   # must print nothing: no untracked files
git status -sb
git status --porcelain=v1
git write-tree
git rev-parse origin/main                  # REVIEW_MAIN_TIP
```

Record the `git write-tree` value as the review-tree fingerprint and the
chosen `REVIEW_BASE`, fetched `REVIEW_MAIN_TIP`, and porcelain status. Do not
hand off while either unstaged tracked changes or untracked files exist: stage
intentional content first and leave unrelated work for a separate ship unit.
This makes any later workspace edit change the clean-workspace checks instead
of relying on porcelain path states to fingerprint file contents. Do **not**
create a temporary review commit. Empty tree and no commits to land may skip
with a one-line reason.

#### Stop and hand off to the user

Use this direct message, adapted only for paths or a non-latest base:

```text
提交前审查已就绪。请在新终端打开当前仓库，使用你选择的终端 Agent
以 `git diff --cached <recorded-review-base-sha>` 为入口只审查完整最终改动，
不要修改文件。交接时请把占位符换成已记录的 REVIEW_BASE SHA。
完成后请把审查结果发回；收到结果前我不会继续提交。
```

Then **stop**. Do not run a reviewer while waiting, do not select an Agent for
the user, and do not interpret silence as approval.

#### Handle the returned result

| User result | Action |
|-------------|--------|
| Explicitly reports no actionable issues / review passed | Fetch `origin` again, then recheck the tree fingerprint, clean workspace, review base, and recorded main tip; if unchanged, proceed to C2 |
| Returns findings | Assess and fix them, re-run relevant tests, stage the intended final tree again, record a new fingerprint/base, and stop for another user-run terminal review |
| Result is ambiguous | Ask whether the external review found any actionable issue; do not infer pass |
| No result yet | Remain at C2-pre; do not commit |
| User explicitly skips review | Record the skip and proceed only within the already-authorized ship chain |

After **any** content change, even a review-driven fix, the old result is stale
and the user must manually review again. Default maximum is **3**
review → fix → handoff cycles; after that, stop and show the remaining findings
instead of committing with known issues.

#### Stale-result and merge-pin rules

- Immediately before accepting the returned result, run `git fetch origin`.
  Require the fetched `origin/main` SHA to equal recorded `REVIEW_MAIN_TIP`;
  otherwise the result is stale even if the merge base did not move.
- **Before C2**, require `git write-tree`, the chosen `REVIEW_BASE`, and
  `git status --porcelain=v1` to match their recorded values. Also require
  `git diff --quiet` to succeed and
  `git ls-files --others --exclude-standard` to print nothing.
- **After C2**, the staged porcelain snapshot normally becomes clean; this is
  valid. Before C3/C4/C5 (including a later `继续`), fetch `origin`, require
  the fetched `origin/main` to equal recorded `REVIEW_MAIN_TIP`, require
  `git rev-parse 'HEAD^{tree}'` to equal the reviewed tree fingerprint, and
  require `git diff --cached --quiet`, `git diff --quiet`, and an empty
  `git ls-files --others --exclude-standard`. A same-tree C2 commit or
  message-only amend does not stale the result.
- After C2/C3, set the merge pin to the pushed tip whose tree matches the
  reviewed fingerprint.
- Before C5, fetch and require PR `headRefOid` to match that merge pin. A moved
  remote head requires another user-run review; never merge an unreviewed tip.
- If the user declined updating a behind branch, review from its recorded merge
  base. Do not mark the result stale from the behind count alone, but do stale
  it when the separately recorded `REVIEW_MAIN_TIP` moves.

#### Allowed skip (must state reason)

| Case | Skip? |
|------|--------|
| Empty tree and nothing to land | Yes — nothing to review |
| User has not returned a result | No — wait, or let the user explicitly skip |
| User explicitly:「跳过审查 / 不审了直接提交 / skip review」 | Yes — record it; still enforce git hygiene and Hard rules |

Gate is a **Mode C quality step** (user-confirmed L2 playbook), not a Hard-rule
rewrite. Pure docs, skill-only changes, small wording, or time pressure are not
automatic exemptions. The current Agent never self-skips.

#### Coach frames

Waiting for the user:

```text
当前模式: C（提交前人工审查）
状态: 等待用户在新终端使用自选 Agent 审查并返回结果
范围: recorded REVIEW_BASE → staged final tree
下一步: 用户返回审查结果；当前 Agent 暂停提交
```

After the result:

```text
当前模式: C（发车）
人工终端 Agent 审查: 通过 / 已修 N 轮后通过 / 跳过（原因）/ 等待复审
下一步: commit（C2）或按 findings 继续修复
```

If C1b (or any other check) still finds work after the user reports a pass and
you edit files, the result is stale: return to C2-pre and wait for another
user-run review before C2.

### C2. Commit (only if user asked)

**Only after C2-pre** (user returned a passing result, allowed skip, or user override).

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

Before drafting the PR body, enforce the repository `AGENTS.md` identity rule:

```bash
git config user.name
git config user.email
git shortlog -sne origin/main | head -20
```

Compare the current identity with recurring core authors in repository history.
Do not change git config. If the current user is not a historical core
developer, the PR body must explicitly state that the change is AI-generated
or AI-assisted.

Always start from `.github/PULL_REQUEST_TEMPLATE.md`, preserve its headings and
fill the relevant sections. For **AkumaRealLabs/ggapi**, AI-generated and
AI-assisted content is allowed and the template's verification record is
optional; this does not waive C1/C2-pre implementation checks. Complete exactly
one `Fork Diff Inventory` choice: either no permanent upstream delta, or the
updated `docs/fork/diff-inventory.md` GG ID.

```bash
PR_BODY_FILE="$(mktemp)"
cp .github/PULL_REQUEST_TEMPLATE.md "$PR_BODY_FILE"
# Fill the copied template; add the required AI disclosure when the identity
# comparison above says the current user is not a historical core developer.
```

If `gh pr create` fails with *No commits between main and docs/…* or blank SHAs
while `git log origin/main..HEAD` shows commits, **do not** open the PR against
upstream. Use the REST API against the team repo with the same filled template:

```bash
gh api repos/AkumaRealLabs/ggapi/pulls \
  -f title='…' \
  -f head="$(git branch --show-current)" \
  -f base='main' \
  -f body="$(<"$PR_BODY_FILE")"
```

Optional after set-default works:

```bash
gh pr create --repo AkumaRealLabs/ggapi --base main \
  --head "$(git branch --show-current)" --title "<title>" \
  --body-file "$PR_BODY_FILE"
```

Do not reuse this fork's template or relaxed AI policy when contributing
**upstream**; Mode G loads the target repository's current rules instead.

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
| `OPEN` + no auto-merge, CLI was transport flake | Retry `gh pr merge` once. REST only if user still wants immediate merge, and **pin the reviewed head**: `gh api -X PUT repos/AkumaRealLabs/ggapi/pulls/<N>/merge -f merge_method=merge -f sha='<mergePin>'` (or `gh pr merge --match-head-commit <oid>`). Before any merge: fetch and compare remote `headRefOid` to the **merge pin** (pushed tip whose tree matches the C2-pre fingerprint); if moved → **stop** and ask the user to run another terminal review; do not land an unreviewed tip |
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
cd web && bun run typecheck
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
| `web/**` | Prefer upstream structure/component APIs, then replay only required GG-004/006/007 capability deltas |
| Removed frontend paths (`web/ggapi`, `web/classic`) | Keep removed; do not restore GG-005 or fork-only visual work |

3. Build/test.
4. Restore inventory rows to `active` (or update summary).
5. Update baseline SHA/date.
6. Compare `web/` with upstream and confirm remaining deltas belong to active inventory rows.

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
6. **After tag push — wait for GHCR before claiming 发版完成** (expect ~8–15 min on GitHub-hosted runners; lesson: `v1.0.0-rc.22.1`):

```bash
REL_TAG=v1.0.0-rc.22.1   # the tag you just pushed
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
# docker pull ghcr.io/akumareallabs/ggapi:v1.0.0-rc.22.1
```

7. Minimum regression:

- Login + API token
- Main inference path + billing
- Top-up/quota changes if customized
- All inventory `active` regression points
- Watch logs for quota saturation / auth errors

```bash
# Local bare binary (not default CI path): build the official frontend before Go
make build-web
# Dockerfile already builds web/dist before Go
```

Confirm Dockerfile still builds and embeds official `web/dist` on the image path.
Do not run production deploy commands without explicit user request and environment confirmation.
Chained「…→ merge → 发版」from Mode C: see **C0** — explicit merge verb required before C5; Mode F only on `origin/main` tip (never treat 发版 alone as merge auth).

---

## §G — Contribute back upstream

1. Search upstream issues/PRs first.
2. Prefer changes that reduce fork drift.
3. Branch from a clean cherry-pick or minimal patch against upstream if needed.
4. After `git fetch upstream`, read the target's current PR template,
   `AGENTS.md` / `CONTRIBUTING*` (when present), and workflow definitions from
   `upstream/main`. Follow those current requirements, including any human
   authorship, understanding, local validation, proof-of-work, or AI rules.
   Never inherit AkumaRealLabs/ggapi's relaxed AI policy into upstream.
5. Query GitHub's remote runtime/policy APIs; a Git tree alone cannot show
   whether a workflow is disabled or which protection/ruleset checks apply:

```bash
gh api repos/QuantumNous/new-api/actions/workflows --paginate \
  --jq '.workflows[] | {id:.id,name:.name,path:.path,state:.state}'
gh api repos/QuantumNous/new-api/branches/main/protection
gh api repos/QuantumNous/new-api/rulesets --paginate \
  --jq '.[] | {id:.id,name:.name,target:.target,enforcement:.enforcement,conditions:.conditions}'
gh api repos/QuantumNous/new-api/rules/branches/main
```

   Use workflow `state` to distinguish `active` from disabled definitions.
   Inspect classic protection `required_status_checks` plus effective branch
   rules for required contexts/reviews. A protection 404 only means no classic
   protection is visible; it does **not** skip the ruleset/effective-rules
   queries. Stop on authentication/permission ambiguity instead of assuming no
   policy.
6. Security: **no** public issue — follow `.github/SECURITY.md`.
7. After upstream merges: remove fork patch, set inventory status `upstreamed`, delete dead code in a ggapi PR.

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
