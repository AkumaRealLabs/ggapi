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

Ship one coherent change on a short-lived branch; it reaches `main` only through the Mode C local merge.

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
   - Add GG-004/007 capabilities directly to the matching official feature modules (Bun, `t('English key')`, **i18n-translate**).
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
ran full-tree and it failed. Fix findings on touched files before C2.

Manual smoke when behavior is user-visible: login/token, one chat path, admin page (`make dev` or `make dev-web` + API).

### B4. Diff-inventory draft (if permanent)

Before asking to commit/merge, prepare an entry for `docs/fork/diff-inventory.md`:

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

## §C — Ship: commit → local merge into main → push main

### Goal

Land work on `origin/main` by merging a short-lived topic branch into local
`main` and pushing `main` to `origin`. This repository has a single
maintainer, so routine work has **no PR step** (skill 4.0.0+). The only PRs
are Dependabot's (C6) and contributions to upstream (Mode G).

### C0. Chained ship authorization (one user message)

When the user **in one message** stacks several Mode C/F verbs, treat that as
**one authorization chain** — run the named steps **in order** in the same
session without re-asking for each sub-step already listed.

| User phrase (examples) | Authorized steps |
|------------------------|------------------|
| 提交 / commit / 最终提交 | C2 only |
| 合并 / 推送 / 推送合并 / 合入 / push（**无**提交词） | **C3 → C4 → C5** on **existing** commits. If worktree dirty → **stop**, ask for 提交 (do **not** invent C2) |
| 提交并合并 / 提交并推送 | C2 → C3 → C4 → C5 |
| 推分支 / 备份分支 | **C-backup only**: push the topic branch; does **not** land on `main` |
| …→ 合并 → **发版** / 合并后打 tag | Same as merge chain **plus** Mode F only after C4 pushed `main` |
| **发版** / tag / 打 tag / GHCR alone | Mode **F only** on already-landed `origin/main`. **Does not** authorize C3–C5 |

Rules:

- **Missing verb = not authorized.** 「只提交」does **not** imply merge or 发版.
- **Merge wording ≠ commit.** 「合并」「推送」alone never run C2 on a dirty tree; only land **already committed** work.
- **发版 ≠ merge.** 「提交并发版」without 合并/推送 does **not** open C4; either the change is already on `origin/main`, or stop and ask for merge authorization.
- Routine work still happens on a **topic branch**; the only commit made on `main` is the C4 merge commit.
- Still **never** `git push upstream`; never force-push `main`.
- Hard-risk ops (force-push, `reset --hard`, production deploy beyond tag/GHCR)
  still need their **own** explicit confirm even inside a chain.
- Mode F only when local `HEAD == origin/main` after fetch.

Coach one-liner after parsing:

```text
当前模式: C（链式发车）[→ F]
本句授权: 提交 / 合并 / 发版（勾选实际出现的）
下一步: <first concrete command>
```

### C-continue. 「继续」when a ship unit is already in flight

User phrases: `继续` / `接着` / `/ggapi-fork 继续` **without** new feature/fix
scope.

**Detect stage first** (preflight: branch, dirty tree, commits not on
`origin/main`, latest `main` CI run).

**Resume rule:** **C0 still wins** — `继续` never invents missing verbs
(提交 / 合并 / 发版).

| Current stage | What 「继续」means | Do **not** |
|---------------|-------------------|------------|
| Mode B coding mid-feature, dirty or unfinished | Resume **B** (implement / verify) | Jump to commit without 提交 verb |
| Ready to ship, dirty tree, no commit auth yet | Coach next Mode C step; **stop** for 提交 if dirty | Invent C2 |
| Commits on topic branch, not on `origin/main` | **Coach only:** next is C3; land **only** if the chain already has 合并/推送 **or** the user says it now. Prior **提交 alone** ≠ merge | Silent C4 |
| Landed; `main` CI running or failed | Report CI; on failure coach a forward-fix branch (needs new 提交/合并) | Force-push or rewrite `main` |
| On `main`, 发版 intent | Mode **F** only if 发版 authorized; else ask | |

**Lesson (PR #26):** once work has landed, `/ggapi-fork 继续` = Mode C
post-ship (CI / cleanup), **not** more product code unless the user names a new
defect.

### C1. Pre-flight

```bash
git branch --show-current   # must NOT be main
git status -sb
git fetch origin
git log --oneline origin/main..HEAD
git diff origin/main...HEAD --stat
```

If currently on `main` with commits that are not merge commits from C4:
**stop**. Move them to a topic branch:

```bash
git branch <prefix>/<topic>    # pointer to current commit
git reset --hard origin/main # ONLY if user confirms; destructive to main tip local
git checkout <prefix>/<topic>
```

Prefer safer alternative if unsure: ask user before any hard reset.

### C1b. Frontend / embed ship checks (when `web/`, Docker, makefile, or release wiring is touched)

Run **before C2** for frontend/embed/Docker/makefile/release changes. If checks force code edits, finish those edits and re-run the affected checks before committing.

| Check | Why |
|-------|-----|
| `web/ggapi` and `web/classic` stay absent; `theme.frontend` remains only upstream's retired-option compatibility path | GG-005 must not be silently restored |
| `makefile` + **`Dockerfile`** (+ optional `release.yml` if dispatching binaries) build `web/dist` before Go embed | Missing official frontend dist breaks GHCR/bare binaries |
| Protected branding: `index.html` title/meta and logo accessible name stay **New API** / QuantumNous policy | Fork features do not remove protected identity |
| New i18n keys live under locale `translation`; all shipped locales (incl. **zh-TW**) have key parity | Orphan top-level keys + missing locales caused review findings |
| `web/`: **typecheck** when TS touched; **lint ship-unit paths** with `bunx oxlint -c .oxlintrc.json <paths>` | Full-tree `bun run lint` may be baseline-noisy; require clean **touched** files only |

Canonical policy: `docs/fork` SOP §3.5; inventory **GG-005** is dropped.

### C2. Commit (only if user asked)

Follow repo commit rules:

- `git status` / `git diff` / `git log` style analysis
- Stage intentional files only
- HEREDOC commit message, why-focused
- Never update git config; never skip hooks

If change is permanent fork delta, include the inventory update on the same
topic branch so it lands with the code.

### C3. Pre-merge checks (only when 合并/推送 is authorized)

No CI runs before landing, so the local checks are the gate:

1. Make sure the B3 checks for the ship unit (touched Go packages, `web/`
   checks, C1b) passed on the current tip. Re-run any check whose inputs
   changed since it last passed, and keep the commands for the report.
2. Bring in the latest `origin/main`:

```bash
git fetch origin
git merge-base --is-ancestor origin/main HEAD || git merge origin/main
```

If `origin/main` had to be merged in, resolve conflicts per §E and re-run the
affected checks. Never `reset --hard` to get past a conflict.

### C4. Merge into main and push

```bash
TOPIC=$(git branch --show-current)
git checkout main
git pull --ff-only origin main
git merge --no-ff --no-edit "$TOPIC"
git push origin main           # never --force
```

- `git pull --ff-only` fails → local `main` has commits that are not on
  `origin/main`: **stop**, troubleshooting §2.
- Push rejected (non-fast-forward) → `origin/main` moved after C3: run
  `git pull --no-rebase origin main` on `main`, re-run the affected checks,
  push again. Never force.
- The merge commit is signed like any other commit; an SSH agent refusal
  means the key is locked — ask the user to unlock it, never bypass signing.

### C5. Clean up and watch `main` CI

```bash
git branch -d "$TOPIC"
# only if the branch was backed up with C-backup:
git ls-remote --exit-code --heads origin "$TOPIC" >/dev/null && git push origin --delete "$TOPIC"
git remote prune origin
```

Pushing `main` triggers `ci.yml` (GG-003). Find the run for the pushed SHA and
wait for it:

```bash
MAIN_SHA=$(git rev-parse HEAD)
gh run list --repo AkumaRealLabs/ggapi --workflow=ci.yml --branch main \
  --event push --limit 5 --json databaseId,headSha
# take the run whose headSha == $MAIN_SHA, then:
gh run watch <run-id> --repo AkumaRealLabs/ggapi --exit-status
```

| Result | Coach action |
|--------|--------------|
| `success` | Report landed SHA + CI green |
| `failure` | Do **not** claim done. `gh run view <id> --log-failed`; separate known upstream failures (same code as `upstream/main`) from fork regressions; fork regression → forward-fix topic branch (needs 提交/合并) |
| No run for the SHA | Check `ci.yml` still has the `push` trigger on `main`; do not borrow another run's result |

C5 leaves you on clean `main`. Next work → new branch from latest
`origin/main`.

### C-backup. Push a topic branch without landing (only on 推分支)

```bash
git push -u origin HEAD
```

The branch on `origin` is only a backup; no CI runs and nothing reaches
`main`. C5 deletes it after the branch lands.

### C6. Dependabot PRs (the only routine PRs)

Dependabot opens PRs against `main` (`.github/dependabot.yml`). Act only on a
PR the user names (例「合并 #47」「关掉 #47」):

```bash
gh pr checks <N> --repo AkumaRealLabs/ggapi
gh pr merge <N> --repo AkumaRealLabs/ggapi --merge --delete-branch \
  --match-head-commit "$(gh pr view <N> --repo AkumaRealLabs/ggapi --json headRefOid -q .headRefOid)"
```

`gh pr merge` can exit non-zero on a transport error after GitHub already
merged — verify `state` before retrying (troubleshooting §23). Always pass
`--repo AkumaRealLabs/ggapi`; this clone's `gh` default may be upstream (§17).
After a merge: `git checkout main && git pull --ff-only origin main`.

### Decision table (teach the user)

| Situation | Do this |
|-----------|---------|
| Finished coding on `feat/x` | 提交 → 合并 (C3 → C4 → C5) |
| Want code on `main` | Local `git merge --no-ff` + `git push origin main`; no PR |
| Docs-only on `docs/…` | Same |
| Sync upstream branch | Same: `sync/…` → local merge → push `main` (§D7) |
| Emergency production break | `hotfix/…` branch, same landing path |
| Back up unfinished work | 推分支 (C-backup) |
| Contribute to upstream | Mode G; the user files the upstream PR |

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

Record range for the report and the sync commit message:

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

### D7. Land the sync on main

After D5/D6 pass and the user says 合并/推送, land `sync/upstream-YYYYMMDD`
exactly like Mode C **C3 → C4 → C5** (local `git merge --no-ff` into `main`,
`git push origin main`, watch `main` CI). Commit the upstream merge on the sync
branch as `chore(sync): merge upstream/main @ <short-sha>`.

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
| `web/**` | Prefer upstream structure/component APIs, then replay only required GG-004/007 capability deltas |
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
Chained「…→ 合并 → 发版」from Mode C: see **C0** — explicit merge verb required before C4; Mode F only on `origin/main` tip (never treat 发版 alone as merge auth).

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
7. After upstream merges: remove fork patch, set inventory status `upstreamed`, delete dead code on a ggapi topic branch landed via Mode C.

Never push ggapi-only branding or private config to upstream.

---

## §I — Skill self-upgrade（可控）

完整策略见 `references/self-upgrade.md`。此处仅执行摘要：

1. **分级：** L0 只诊断 → L1 安全小补 → L2 扩展能力 → L3 动硬规则。
2. **可主动：** 文档漂移、实战证明 skill 错、用户要求升级/自检。
3. **不可乱来：** 业务分支顺手大改 skill、削弱 Hard rules、自动 commit/push、无文档依据发明流程。
4. **L1：** 一句话告知后可改文件 + CHANGELOG + bump version；**不**自动提交。
5. **L2/L3：** 先出方案，用户确认后再改；L3 必须独立说明规则变更。
6. **入库：** `docs/ggapi-fork-*` 分支 → 用户说提交后 commit → 用户说合并后本地合入 `main` 并推送 → 删分支。
7. **真相源：** `docs/fork/*` > skill > 会话临时话。
8. **对齐清单：** 升级后核对 GG-002 摘要与 `skill_version`；提及的永久差异（GG-003/004/005 等）须与 inventory 一致。
9. **触发词：** 「自提升」与「自升级 / 升级 skill / Mode I」同等进入本模式。
