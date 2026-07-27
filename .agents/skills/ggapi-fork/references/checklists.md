# ggapi Fork Checklists

Copy into PR bodies or walk verbally with the user. Keep items checkable.

---

## §Pre-PR (feature / fix / docs)

### Git hygiene

- [ ] Not on `main` (topic branch name matches type: `feat/` `fix/` `docs/` `chore/` `hotfix/`)
- [ ] Branched from latest `origin/main` (or rebased/merged main if long-lived)
- [ ] `git status` clean except intended commits
- [ ] No secrets, `.env`, customer data, production connection strings
- [ ] No accidental upstream branding removals (**New API** / QuantumNous protected identity intact)

### Implementation

- [ ] Customization prefers config / independent package over hotspot patch
- [ ] Touched hotspots (quota/auth/relay/model locks)? Dual review planned
- [ ] Billing changes: read `pkg/billingexpr/expr.md`; full pre-consume→settle path checked
- [ ] DB changes: SQLite + MySQL + PostgreSQL considered
- [ ] JSON via `common.Marshal/Unmarshal*` (not raw `encoding/json` calls)
- [ ] Frontend product work in **`web/ggapi`** (not only `web/default`); typecheck on shell when available; **lint ship-unit paths** (not full-tree baseline)
- [ ] Frontend: i18n keys via **i18n-translate** skill; keys under `translation`; all locales incl. **zh-TW**

### Verification

- [ ] Scoped `go test` for touched packages
- [ ] Frontend: ggapi/default → `bun run typecheck` + path-scoped oxlint on **changed files**; classic → path-scoped eslint/prettier (no typecheck); full-tree lint optional if baseline-noisy
- [ ] Manual smoke for user-visible behavior
- [ ] **C1b** third-shell/embed checks (incl. path-scoped lint on touched UI) done **before** C2-pre when GG-005 wiring touched; any resulting edits require another user-run review
- [ ] **User-run terminal-agent gate (C2-pre):** current Agent staged/fingerprinted the combined final tree, required no unstaged/untracked content, stopped, and asked the user to review it in a new terminal with their preferred Agent
- [ ] User explicitly returned a clean result, or an explicit skip/override is recorded; the current Agent did not invoke any reviewer or infer pass from silence
- [ ] Findings were fixed and handed back for another user-run review after every content-changing fix batch
- [ ] Behind-main branch: update preferred; if declined, review diff uses recorded `git merge-base HEAD origin/main`, never latest main directly
- [ ] Gate not **stale before C2**: immediately fetched `origin`; `git write-tree`, clean workspace, porcelain snapshot, `REVIEW_BASE`, and recorded `REVIEW_MAIN_TIP` still match
- [ ] Gate not **stale after C2**: fetched `origin`; current `origin/main` equals recorded `REVIEW_MAIN_TIP`; `HEAD^{tree}` equals the reviewed fingerprint; index/worktree/untracked state is clean. Normal staged → clean porcelain transition and message-only amend of the same tree remain valid
- [ ] Ambiguous `继续`: stage detected (C-continue) — post-PR open → CI/merge coach; never invent push/PR/merge verbs; not silent Mode B

### Fork governance

- [ ] Permanent delta? `docs/fork/diff-inventory.md` updated (or explicitly N/A)
- [ ] Inventory risk level honest (`high` if billing/auth/relay core **or** third-shell wiring GG-005)
- [ ] Regression points written so next sync can re-verify

### Third shell / embed (GG-005 — when `web/ggapi`, theme, embed, Docker, makefile, or release touched)

- [ ] Backend + shipped admin surfaces accept `theme.frontend=ggapi`: `web/ggapi` + `web/default` (enum / select / normalize) **and** classic theme helpers that write the option
- [ ] `makefile` / **`Dockerfile`** build ggapi dist (image path); `release.yml` only if dispatching bare binaries
- [ ] Local bare `go build` / tag binary: `make build-all-web` (all three embed trees; not ggapi-only)
- [ ] Protected title/meta/logo identity not replaced with bare fork product name
- [ ] Locale key parity across en/zh/fr/ja/ru/vi/zh-TW for new strings

### Ship

- [ ] `git push -u origin HEAD` (branch only)
- [ ] `gh` targets **AkumaRealLabs/ggapi** (`gh repo view` / `gh repo set-default`)
- [ ] PR base = `main` on team repo (not QuantumNous/new-api)
- [ ] PR describes what / why / how tested
- [ ] No `git push upstream`
- [ ] If UI strings added: **i18n-translate** on active shell for all locales including **zh-TW**; `i18n:sync` report clean **and** no English left in `zh-TW` (sync treats zh-TW as non-Latin; if unsure, run skill `find-untranslated` path). Classic strings → classic `i18n:*` scripts, not default/ggapi locale JSON.
- [ ] If CI/workflows: existing jobs still use GitHub-hosted `ubuntu-latest` Linux amd64 unless an intentional, inventoried change says otherwise (GG-003)
- [ ] If update-check: server proxy + PAT write-only (GG-004); no PAT in frontend
- [ ] If third shell / theme wiring: GG-005 checks above

---

## §Pre-PR (upstream sync branch)

- [ ] Branch name `sync/upstream-YYYYMMDD`
- [ ] No feature/fix commits mixed in
- [ ] `git fetch upstream` done; PR cites upstream short SHA
- [ ] Conflicts resolved per inventory + Mode E (no whole-tree ours/theirs)
- [ ] `AGENTS.md` still contains ggapi 二开 section if it was present
- [ ] Inventory meta: baseline commit + date updated
- [ ] `needs-rebase` rows restored or still marked with reason
- [ ] No large `web/ggapi` skin refactors mixed into the sync commit
- [ ] Tests: `common` `service` `model` `relay` (and frontend if needed)
- [ ] Smoke: login/token, chat+billing, admin basic page, inventory `active` items
- [ ] Push to **origin** only; PR title `sync: merge upstream/main @ <sha>`

---

## §Post-sync (after sync PR merges)

- [ ] Local `main` pulled
- [ ] Sync remote branch deleted
- [ ] Inventory baseline matches what actually landed
- [ ] Any emergency fix during sync registered as GG-xxx
- [ ] If `web/default` gained user-visible features: follow-up `chore/port-default-*` planned or opened (GG-005)
- [ ] Team notified if `high` behavior changed
- [ ] Optional: note next sync window

---

## §Release

- [ ] DB backup completed
- [ ] Version / tag strategy recorded: `v<upstream-baseline>.N` (e.g. `v1.0.0-rc.20.1`); formal tags only on `origin/main` tip
- [ ] Default product: tag → **GHCR** + **Release metadata** (`docker-build.yml`); `:latest` only after tip recheck; **not** Docker Hub `calciumion/new-api`
- [ ] After tag push: list/watch run **for that tag** (`gh run list --branch "$REL_TAG"` + `gh run watch --exit-status`); only then claim 发版完成; on failure see §24
- [ ] Bare binary assets: only if user wants — `release.yml` **manual dispatch + required tag**
- [ ] Migrations reviewed for target DB
- [ ] Dockerfile builds **ggapi** (+ default + classic) for image path; local bare binary still needs `make build-all-web` first
- [ ] Regression: login/token
- [ ] Regression: main model path + billing correctness
- [ ] Regression: quota/top-up paths if customized
- [ ] Regression: every inventory `active`「回归要点」
- [ ] Watch error logs / quota_saturation after deploy
- [ ] Rollback owner known

---

## §New permanent customization (before merge)

Fill before landing:

```markdown
| ID | 类型 | 路径/范围 | 差异摘要 | 原因 | 上游冲突风险 | 回归要点 | 状态 | 负责人 | 引入日期 | 关联分支/PR |
|----|------|-----------|----------|------|--------------|----------|------|--------|----------|-------------|
| GG-xxx | feature/patch/… | `path/` | 相对上游多了/改了什么 | 为何不能只靠配置 | low/medium/high | 如何验证 | active | name | YYYY-MM-DD | `branch` #n |
```

Risk cheat sheet:

| Risk | Typical paths |
|------|----------------|
| low | `docs/fork/`, own `pkg/`, new channel dir |
| medium | router registration, settings extension, `AGENTS.md` fork section, feature entrypoints |
| high | `service/*quota*`, `model/` tx/locks, `relay/`, auth middleware, `pkg/billingexpr/` |

---

## §Weekly health (optional coach prompt)

- [ ] `git fetch upstream` and skim new commits
- [ ] Behind count: `git rev-list --count main..upstream/main`
- [ ] Inventory date not stale after last sync
- [ ] No long-lived topic branches abandoned without PR
- [ ] No open sync PR older than the agreed window without update

---

## §Skill self-upgrade (Mode I)

### Before editing skill files

- [ ] Read `references/self-upgrade.md`
- [ ] Level decided: L0 / L1 / L2 / L3
- [ ] L2/L3: user confirmed written plan
- [ ] Change is grounded in `docs/fork/*` or a verified session fix
- [ ] Not weakening Hard rules / branding / billing safety
- [ ] Not mixing with unrelated feature work (unless user insisted)

### After editing skill files

- [ ] `metadata.skill_version` bumped (semver per self-upgrade.md)
- [ ] `references/CHANGELOG.md` entry added
- [ ] Mode router still matches workflows
- [ ] Chinese coaching defaults intact
- [ ] No auto-commit / auto-push language introduced
- [ ] User-run terminal-agent gate (C2-pre) still documented if ship-path changed; current Agent never runs the reviewer
- [ ] User shown summary; commit only if they asked

### Skill PR

- [ ] Branch `docs/ggapi-fork-…` from latest main
- [ ] PR describes level + why + what not changed
- [ ] GG-002 inventory note still accurate (update if role changed)
