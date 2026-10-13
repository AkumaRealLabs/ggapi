# ggapi Fork Troubleshooting

Use when the user is stuck, git state is messy, or a workflow command fails.
Diagnose → explain in plain language → propose the **safest** fix → confirm
before destructive actions.

**对用户解释与下一步默认简体中文**；`git` 命令与错误原文可保留英文便于对照。

---

## Quick triage

```bash
git remote -v
git branch -vv
git status -sb
git stash list
git log --oneline -5
git fetch origin 2>&1
git fetch upstream 2>&1
```

| Symptom | Jump to |
|---------|---------|
| `upstream` missing / fetch fails | §1 |
| Accidentally committed on `main` | §2 |
| Unsure how work reaches `main` | §3 |
| Push rejected / non-fast-forward (branch or `main`) | §4 |
| Merge conflicts during sync | §5 |
| Lost fork customization after sync | §6 |
| Diverged badly from upstream | §7 |
| Dirty tree blocks branch switch | §8 |
| Dependabot PR will not merge | §9 |
| Tests fail after sync | §10 |
| Billing / quota weirdness after custom patch | §11 |
| Afraid of breaking protected branding | §12 |
| Inventory out of date / unknown baseline | §13 |
| Working on wrong remote or fork | §14 |
| Skill keeps rewriting itself / user fears chaotic upgrades | §15 |
| Skill out of date vs docs/fork | §16 |
| `gh` shows upstream data / wrong repo | §17 |
| CI jobs stuck / not on GitHub-hosted runner | §18 |
| Update check 401/404 on private repo | §19 |
| i18n sync report missingCount on zh-TW | §20 |
| `继续` after commit or landing re-codes instead of merge/CI | §21 + workflows **C-continue** |
| Official frontend / retired theme compatibility / embed | §22 |
| `gh pr merge` (Dependabot PR) EOF / timeout but PR might be merged | §23 |
| Tag pushed but GHCR / docker-build not “done” | §24 |

---

## §1 Remote / upstream problems

**Missing upstream**

```bash
git remote add upstream https://github.com/QuantumNous/new-api.git
git remote set-url --push upstream DISABLED
git fetch upstream
```

**Accidentally can push upstream**

```bash
git remote set-url --push upstream DISABLED
git remote -v   # push URL should show DISABLED
```

**Fetch auth errors to GitHub**
Check network/token/SSH; do not disable TLS or store secrets in repo files.

---

## §2 Commits landed on local `main`

**Preferred recovery** (preserve commits on a topic branch):

```bash
# ensure clean understanding of commits not on origin/main
git log --oneline origin/main..main

git branch docs-or-feat/<topic>    # save pointer
# ONLY with user confirmation:
git fetch origin
git reset --hard origin/main      # local main matches remote
git checkout docs-or-feat/<topic>
```

Then Mode C: 提交/合并 per C0 (C3 → C5).

**If already pushed commits to `origin/main`**
Do **not** force-push `main`. Prefer a revert or forward-fix branch landed via
Mode C.

---

## §3 How work reaches `main` (stuck in decision)

Teach this one-liner:

> 「提交」saves work on the topic branch; 「合并」merges it into local `main` and pushes `origin/main`; 「推分支」only backs the branch up.

No PR is needed for routine work (skill 4.0.0+). A pushed topic branch alone
has not landed.

---

## §4 Push rejected

**Non-fast-forward on topic branch** (someone else pushed / you rebased):

```bash
git fetch origin
git log --oneline HEAD..origin/<branch>
# prefer:
git pull --rebase origin <branch>   # only on private topic branch if team ok with rebase
# or merge:
git pull origin <branch>
```

**Rejected push to `main`** (non-fast-forward after C4)
`origin/main` moved after C3. On `main`: `git pull --no-rebase origin main`,
re-run the affected checks, push again. **Never** force. If GitHub rejects the
push for permissions or a protection rule, stop and ask the user.

**Permission denied to origin**
User/token lacks write access to `AkumaRealLabs/ggapi` — escalate to human, do not switch remotes silently.

---

## §5 Merge conflicts (sync)

1. List conflicts: `git status`
2. Open `docs/fork/diff-inventory.md` — mark impacted rows `needs-rebase`
3. Resolve file-by-file using Mode E table in `workflows.md`
4. `git add` resolved paths
5. `git commit` only to conclude the merge (if merge stopped mid-way)
6. Run scoped tests
7. Restore inventory + baseline SHA

**Anti-patterns**

- `git checkout --theirs .` / `--ours .` on entire tree
- Deleting inventory `active` rows to "make sync green"
- Mixing a product feature into the sync merge commit

---

## §6 Customization disappeared after sync

1. `git log --oneline upstream/main..main` and search for the feature commits
2. `git log -p -- path/to/file` across sync merge
3. Check inventory: was it `active`? Was conflict resolved wrong?
4. Recover with targeted `git show <sha>:path` / cherry-pick / re-apply minimal patch
5. Add regression note so it does not vanish next sync

---

## §7 Huge divergence / fear of merge

```bash
git fetch upstream
git rev-list --left-right --count main...upstream/main
git log --oneline main..upstream/main | head
git log --oneline upstream/main..main | head
```

Plan:

1. Inventory `high` items first
2. Prefer smaller intermediate syncs if behind many months
3. Still use `sync/upstream-YYYYMMDD` + merge, then land via Mode C (C3 → C5)
4. Do not rewrite shared history with rebase of `main` onto upstream

---

## §8 Dirty working tree

```bash
git status -sb
```

Options (ask user preference):

- Finish and commit on current branch
- `git stash push -u -m "wip"` then switch (remind to `stash pop`)
- Discard only if user explicitly wants (`git restore` / clean) — confirm first

Never `git clean -fdx` without explicit approval.

---

## §9 Dependabot PR will not merge

| Cause | Fix |
|-------|-----|
| `gh` not auth | `gh auth status`; user logs in |
| Wrong repo | Pass `--repo AkumaRealLabs/ggapi` (§17) |
| Checks failing | `gh pr checks <N>`; tell the user; merge only if they still want it |
| Conflicts / outdated branch | Ask the user whether to comment `@dependabot rebase` or close the PR |
| `--match-head-commit` rejected | Head moved; re-read `headRefOid` and checks before retrying |

---

## §10 Tests fail after upstream merge

1. Identify whether failure is upstream-known or fork patch
2. If fork patch on hotspot: re-read upstream change; re-apply with new APIs
3. Prefer fixing fork patch over disabling tests
4. Billing failures: read `pkg/billingexpr/expr.md` + `common/quota_math.go` invariants
5. DB failures: check SQLite/MySQL/PostgreSQL dialect assumptions

---

## §11 Billing / quota after custom work

Stop and verify full chain:

```text
validate bounds → EstimateBilling/OtherRatios → QuotaFrom*Checked
→ pre-consume → settle/refund → consume log (+ quota_saturation admin_info)
```

Rules of thumb:

- No negative charges from overflow
- No bare `int(float64…)` quota casts
- Multipliers bounded; unsigned fields need upper bounds
- Dual review before merge

If unsure, **do not ship** — flag for human billing owner.

---

## §12 Branding / license anxiety

Protected: **new-api** project identity, **QuantumNous** attribution, AGPLv3
obligations. Safe: additive docs, fork SOP, non-destructive display layers
(`branding-safe` type) that **do not** strip upstream credits.

If user asks to remove upstream branding: **refuse** per AGENTS.md Project Governance.

---

## §13 Inventory / baseline unknown

```bash
git fetch upstream
git rev-parse --short upstream/main
git merge-base main upstream/main | xargs git rev-parse --short
```

Update `docs/fork/diff-inventory.md` meta:

- 基准 upstream commit
- 最近更新日期
- Note if `VERSION` empty

Re-scan `git diff --stat upstream/main...main` for unregistered permanent paths.

---

## §14 Wrong remote / forked the wrong place

If `origin` is not `AkumaRealLabs/ggapi`:

1. Show `git remote -v` to user
2. Do not `push` until they confirm correct origin
3. Fix remote URL only with explicit approval

---

## §15 Skill self-upgrade felt chaotic

1. Open `references/self-upgrade.md` and `references/CHANGELOG.md`.
2. Check last entries: were they L1 noise or L3 without consent?
3. Recovery: `git log -- .agents/skills/ggapi-fork` / revert the skill-only commit.
4. Remind gates: no auto-commit; L2/L3 need plan; Hard rules frozen without explicit L3.
5. If agent upgraded mid-feature: separate skill commit or revert skill hunks from the feature branch.

## §16 Skill teaches steps that contradict docs/fork

1. **Trust docs** for what you do now.
2. Diff the relevant section vs skill Mode.
3. Mode I **L1**: patch skill to match docs (do not “fix” docs unless SOP change is intended).
4. Bump version + CHANGELOG; ship via docs branch when user wants it on main.

---

## §17 `gh` targets the wrong repo (upstream)

**Symptom:** `gh pr` / `gh run` commands list upstream PRs or runs, or report
*No commits between main and …* for a branch that has commits.

**Cause:** `gh` default repo is `QuantumNous/new-api` (upstream), not
`AkumaRealLabs/ggapi`.

```bash
gh repo view --json nameWithOwner -q .nameWithOwner
# bad:  QuantumNous/new-api
# good: AkumaRealLabs/ggapi

gh repo set-default AkumaRealLabs/ggapi
```

Pass `--repo AkumaRealLabs/ggapi` on every `gh` call anyway. Never open PRs
against upstream unless the user is doing Mode G (contribute upstream) on
purpose.

---

## §18 CI not using GitHub-hosted runners / jobs queued forever

**Expected (GG-003):** workflows use:

```yaml
runs-on: ubuntu-latest
```

Linux **amd64 only** (no arm64 multi-arch, no macOS/Windows release, Electron disabled).

| Check | Action |
|-------|--------|
| Workflow still uses `runs-on.group` / self-hosted labels | Stale fork customization or sync regression; restore `ubuntu-latest` and keep GG-003 current |
| Jobs queued | Check GitHub Actions service status, repository Actions permissions, concurrency, and account spending limits rather than runner-group availability |
| Docker build fails | GitHub-hosted runners include Docker; inspect Buildx setup, disk pressure, cache, and workflow logs |
| Architecture drift | Keep release/image jobs on GitHub-hosted Linux x64; do not silently add arm64/macOS/Windows matrices |

---

## §19 Private-repo update check fails (401/404)

**Expected (GG-004):** Root configures:

- `UpdateCheckRepoAPIURL` — e.g. `https://api.github.com/repos/AkumaRealLabs/ggapi/releases/latest`
- `UpdateCheckGitHubToken` — PAT with **Contents: Read** (never returned by GET `/api/option/`)

Browser no longer calls GitHub directly; use `GET /api/option/check_update`.

| Status | Likely cause |
|--------|----------------|
| 404 | Wrong URL, no releases, or token lacks access |
| 401/403 | Missing/expired PAT or wrong scopes |
| Token field always empty after save | Normal — suffix `Token` is stripped from GetOptions; leave blank to keep existing |

---

## §20 i18n missingCount on zh-TW (or other locales)

`bun run i18n:sync` may copy English into `zh-TW.json` without real translations.
When adding keys via `add-missing-keys` style scripts, **include every locale file
the project ships** (`en`, `zh`, `zh-TW`, `fr`, `ja`, `ru`, `vi` as applicable).

```bash
# Official frontend:
cd web && bun run i18n:sync
# inspect src/i18n/locales/_reports/_sync-report.json
# missingCount and untranslatedCount should be 0 for locales you claim complete
```

Hand off bulk translation work to **i18n-translate** skill; never leave zh-TW as
English-only for user-facing strings you just introduced.

Also reject **orphan top-level keys** outside `translation` in locale JSON (they
do not load as normal `t()` keys). Nest under `translation` and keep key sets
aligned across locales.

---

## §21 「继续」routed wrong (re-coding after commit or landing)

Symptom: ship unit already committed (waiting for 合并) or landed (C4 pushed
`main`), user says `继续`, agent starts new feature edits.
Fix: **C-continue** — wait for 合并 wording, or report `main` CI / C5 hygiene.
Only re-enter Mode B if user names a new defect/scope.

---

## §22 Official single frontend / retired theme compatibility / embed

Canonical: `docs/fork/README.md` §前端, SOP §3.5. Inventory **GG-005** is `dropped`.

### Where frontend changes belong

The only frontend is `web/`. GG-004/007 changes belong in matching official
feature modules and should stay minimal relative to upstream. Do not recreate
`web/ggapi`, classic, multi-theme routing, or fork-only visual redesigns.

### Local commands

```bash
make dev-web
# or: make dev          # API + official frontend
make build-web
```

### Release / Docker binary missing frontend assets

Symptom: tag image or optional bare binary serves a blank dashboard.
Cause: Go embed has no `web/dist`.
Fix: ensure **Dockerfile** builds `web/` before Go for the default **tag → GHCR**
path; if dispatching **`release.yml`** bare binaries, that workflow must also
build `web/`; local bare `go build` needs `make build-web` first.

### Legacy `theme.frontend` values

Upstream retains a compatibility migration that normalizes retired
`theme.frontend` values to `default`. Keep that migration when syncing, but do
not add a selector or treat `ggapi`/`classic` as active runtime themes.

### Branding review findings after frontend work

Do **not** replace protected **New API** / QuantumNous identity in
`index.html` title/meta or logo accessible names with a bare product codename.
Fork functionality can use normal UI copy / SystemName config; protected
metadata stays per AGENTS.md.

### i18n: keys at JSON root, missing in fr/ja/…

See §20. Nest under `translation`; fill all locale files (English fallback is
OK short-term if documented; prefer **i18n-translate** for real copy).

### After merge: where am I?

C5 leaves you on clean `main`. Next feature → new branch from latest
`origin/main`. Do not keep coding on a deleted topic branch.

---

## §23 `gh pr merge` GraphQL EOF / timeout (PR may already be merged)

**Symptom:** `gh pr merge` exits non-zero with GraphQL EOF, connection reset,
deadline exceeded, or similar transport error. Agent or user assumes merge
failed and may re-run merge or leave the branch “stuck.”

**Cause:** Client/API flake; GitHub may have completed the merge. Seen on
internal PRs (e.g. #23) with `--merge --delete-branch`.

**Fix — verify before retry:**

```bash
gh pr view <N> --repo AkumaRealLabs/ggapi \
  --json state,mergedAt,mergeCommit,autoMergeRequest,mergeStateStatus,headRefOid,url
# If GraphQL also fails: gh api repos/AkumaRealLabs/ggapi/pulls/<N>
# state == MERGED / merged:true → success: fetch origin, checkout main, pull, prune
# OPEN + autoMergeRequest / checks pending → wait; do not REST-force; do not tag
# OPEN + no auto-merge after flake → retry merge; REST only with -f sha=<headRefOid>
```

| Observed | Do |
|----------|-----|
| MERGED | `git pull --ff-only origin main` on `main` (C6); **do not** merge again |
| OPEN + auto-merge / merge queue / checks | Poll until MERGED or failure; **never** Mode F on old main |
| OPEN + no auto-merge + transport flake | Retry `gh pr merge`; REST with pinned `sha`/head; check branch protection |
| View + REST both fail | Stop; do not assume merged |
| CLOSED unmerged | User closed without merge — do not invent success |

Always pass `--repo AkumaRealLabs/ggapi` (same wrong-default risk as §17).

---

## §24 Tag pushed — waiting for GHCR (`docker-build.yml`)

**Symptom:** Annotated tag is on `origin`, but coach reports “发版完成” while
image is still building, or user asks whether GHCR is ready.

**Expected:** Tag push triggers **Publish Docker image** (~8–15 min on
GitHub-hosted runners). Product path is GHCR + Release metadata (GG-003 / SOP §5.1), not
bare binary.

```bash
REL_TAG=v1.0.0-rc.22.1   # example — must match the tag you pushed
gh run list --repo AkumaRealLabs/ggapi --workflow=docker-build.yml \
  --branch "$REL_TAG" --limit 5
# Use the run id for this tag (headBranch == REL_TAG / matching headSha), then:
gh run watch <run-id> --exit-status --repo AkumaRealLabs/ggapi
# On success (exit 0):
# docker pull ghcr.io/akumareallabs/ggapi:${REL_TAG}
```

| Status | Action |
|--------|--------|
| `in_progress` / `queued` | Wait or share run URL; **do not** claim 发版完成 |
| `success` and watch exit 0 | Report image tag; optional pull; then regression notes |
| `failure` / watch exit ≠ 0 | `gh run view <id> --log-failed`; fix CI (often Dockerfile/embed §22) before re-tag |
| No run for `$REL_TAG` | Do **not** pick another tag’s green run; confirm tag form, workflow paths, §18 |

Do not re-push the same tag name. If main moved after tag, tag still pins the
old SHA (SOP §5.1 post-push warning) — human decides re-cut.

---

## Emergency "I already ran a dangerous command"

| Already did | Mitigation |
|-------------|------------|
| `reset --hard` lost commits | `git reflog` → recover SHA → branch from it |
| Force-pushed topic branch | Coordinate with anyone who pulled; avoid force on `main` |
| Pushed secrets | Rotate credentials; history purge only with team process |
| Merged bad sync to main | Fix-forward branch landed via Mode C; inventory note; never hard reset or force-push `main` |
| Skill auto-edited and confused you | See §15; discard uncommitted skill diffs or revert commit |

Always show reflog recovery options before declaring data lost:

```bash
git reflog | head -30
```
