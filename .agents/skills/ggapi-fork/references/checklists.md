# ggapi Fork Checklists

Walk verbally with the user before landing. Keep items checkable.

---

## §Pre-merge (feature / fix / docs)

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
- [ ] Frontend work stays in official **`web/`** and is limited to the smallest active GG-004/007 capability delta; **lint ship-unit paths** (not full-tree baseline)
- [ ] Frontend: i18n keys via **i18n-translate** skill; keys under `translation`; all locales incl. **zh-TW**

### Verification

- [ ] Scoped `go test` for touched packages
- [ ] Frontend: from `web/`, run `bun run typecheck` + path-scoped oxlint on **changed files**; full-tree lint optional if baseline-noisy
- [ ] Manual smoke for user-visible behavior
- [ ] **C1b** frontend/embed checks (incl. path-scoped lint on touched UI) done **before** C2; re-run affected checks after any resulting edits
- [ ] Ambiguous `继续`: stage detected (C-continue) — after commit or landing → merge/CI coach; never invent 提交/合并/发版 verbs; not silent Mode B

### Fork governance

- [ ] Permanent delta? `docs/fork/diff-inventory.md` updated (or explicitly N/A)
- [ ] Inventory risk level honest (`high` if billing/auth/relay core)
- [ ] Regression points written so next sync can re-verify

### Official frontend / embed (when `web/`, embed, Docker, makefile, or release is touched)

- [ ] `web/ggapi` and `web/classic` remain absent; GG-005 and runtime multi-theme wiring are not restored
- [ ] `makefile` / **`Dockerfile`** build official `web/dist` (image path); `release.yml` does the same only when dispatching bare binaries
- [ ] Local bare `go build` / tag binary: run `make build-web` first
- [ ] Protected title/meta/logo identity not replaced with bare fork product name
- [ ] Locale key parity across en/zh/fr/ja/ru/vi/zh-TW for new strings

### Ship

- [ ] User said 合并/推送 (C0); otherwise stop after the commit
- [ ] C3: checks pass on the current tip; latest `origin/main` merged in and affected checks re-run
- [ ] C4: `git merge --no-ff` into local `main`; `git push origin main` without `--force`
- [ ] C5: topic branch deleted (remote too if backed up); `main` CI run for the pushed SHA watched to completion
- [ ] No `git push upstream`
- [ ] If UI strings added: **i18n-translate** from `web/` for all locales including **zh-TW**; `i18n:sync` report clean **and** no English left in `zh-TW` (sync treats zh-TW as non-Latin; if unsure, run skill `find-untranslated` path)
- [ ] If CI/workflows: `ci.yml` keeps its `push` trigger on `main`; existing jobs still use GitHub-hosted `ubuntu-latest` Linux amd64 unless an intentional, inventoried change says otherwise; no anti-slop auto-close gate is restored for **AkumaRealLabs/ggapi** PRs (AI-generated and AI-assisted contributions are allowed locally; upstream policy remains independent) (GG-003)
- [ ] If update-check: server proxy + PAT write-only (GG-004); no PAT in frontend
- [ ] If frontend/embed wiring changed: official single-frontend checks above

---

## §Pre-merge (upstream sync branch)

- [ ] Branch name `sync/upstream-YYYYMMDD`
- [ ] No feature/fix commits mixed in
- [ ] `git fetch upstream` done; sync merge commit cites upstream short SHA
- [ ] Conflicts resolved per inventory + Mode E (no whole-tree ours/theirs)
- [ ] `AGENTS.md` still contains ggapi 二开 section if it was present
- [ ] Inventory meta: baseline commit + date updated
- [ ] `needs-rebase` rows restored or still marked with reason
- [ ] `web/` conflicts use upstream structure first; remaining deltas map only to active inventory rows
- [ ] Tests: `common` `service` `model` `relay` (and frontend if needed)
- [ ] Smoke: login/token, chat+billing, admin basic page, inventory `active` items
- [ ] Land via C3 → C5; push to **origin** only; sync commit `chore(sync): merge upstream/main @ <sha>`

---

## §Post-sync (after the sync branch lands)

- [ ] Local `main` pulled
- [ ] Sync branch deleted (remote too if backed up)
- [ ] Inventory baseline matches what actually landed
- [ ] Any emergency fix during sync registered as GG-xxx
- [ ] `web/ggapi`, classic, and multi-theme wiring remain removed (GG-005 stays dropped)
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
- [ ] Dockerfile builds official `web/dist` for image path; local bare binary still needs `make build-web` first
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
- [ ] No long-lived topic branches left unmerged
- [ ] No sync branch older than the agreed window left unmerged

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
- [ ] User shown summary; commit only if they asked

### Skill landing

- [ ] Branch `docs/ggapi-fork-…` from latest main
- [ ] Commit message / report states level + why + what not changed
- [ ] GG-002 inventory note still accurate (update if role changed)
