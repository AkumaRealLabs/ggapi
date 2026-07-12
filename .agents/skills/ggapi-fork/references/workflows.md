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

Frontend (if UI/TS touched — run on the shell you edited; keep make at **repo root**):

```bash
(cd web/ggapi && bun run typecheck)     # product shell (usual)
# (cd web/default && bun run typecheck) # only if you intentionally changed default
# lint if project scripts require it for the change
# when ship/embed risk (optional mid-feature); always from repo root.
# Build the shell you actually edited (not only the product shell by habit):
make build-web-ggapi   # if web/ggapi changed
# make build-web       # if web/default changed
# make build-web-classic  # if web/classic changed
# make build-all-web   # before bare go build / full embed verification
```

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
| Invocation | Agent uses **companion / `codex review` CLI**; slash is optional UX for humans. |
| Full unit | Always one combined base→final-tree review; mixed → materialize then `--base origin/main`. |
| Fix owner | **ggapi-fork agent** applies fixes; never claim review will edit code. |
| Round limit | Default **3** full cycles (review → fix → re-review). Then stop and escalate. |
| No silent skip | Skipping requires an allowed case below **and** a one-line reason. Docs/skill changes are **not** auto-exempt. |
| Still need auth | Passing the gate does **not** auto-commit; C2 still needs explicit commit intent. |
| Optional extra | `/codex:adversarial-review` or bundled `/review` only if user asks; not a substitute. |

#### Allowed skip (must state reason)

| Case | Skip? |
|------|--------|
| Empty tree and nothing to land | Yes — nothing to review |
| Codex plugin / CLI / companion unavailable or auth failure | Do **not** pretend passed: stop, §21; **default = 先不提交** until user chooses retry / install / 「跳过审查并提交」 |
| User explicitly:「跳过 Codex / 不审了直接提交」 / skip review | Yes — record in reply; still do normal git hygiene |

Gate is a **Mode C quality step** (user-confirmed L2 playbook), not a Hard-rule rewrite: user may always opt out with clear language; agent never auto-skip docs/skill.

**Not** an allowed unilateral skip: pure docs, skill-only, or “small wording”
changes — including changes to this gate. Same gate unless the user opts out.

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
merge on open alone.

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
6. Minimum regression:

- Login + API token  
- Main inference path + billing  
- Top-up/quota changes if customized  
- All inventory `active` regression points  
- Watch logs for quota saturation / auth errors  

```bash
# Default: after tag push, pull GHCR (package may be private — docker login ghcr.io)
# docker pull ghcr.io/akumareallabs/ggapi:v1.0.0-rc.20.1

# Local bare binary (not default CI path): embeds need all three dist trees
make build-all-web
# Dockerfile already builds default + classic + ggapi (builder-ggapi) then go
```

Confirm Dockerfile still builds **ggapi** (and other embedded shells) on image path (GG-005 / #7).  
Do not run production deploy commands without explicit user request and environment confirmation.

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
