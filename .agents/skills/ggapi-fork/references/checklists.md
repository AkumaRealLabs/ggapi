# ggapi Fork Checklists

Copy into PR bodies or walk verbally with the user. Keep items checkable.

---

## §Pre-PR (feature / fix / docs)

### Git hygiene

- [ ] Not on `main` (topic branch name matches type: `feat/` `fix/` `docs/` `chore/` `hotfix/`)
- [ ] Branched from latest `origin/main` (or rebased/merged main if long-lived)
- [ ] `git status` clean except intended commits
- [ ] No secrets, `.env`, customer data, production connection strings
- [ ] No accidental upstream branding removals

### Implementation

- [ ] Customization prefers config / independent package over hotspot patch
- [ ] Touched hotspots (quota/auth/relay/model locks)? Dual review planned
- [ ] Billing changes: read `pkg/billingexpr/expr.md`; full pre-consume→settle path checked
- [ ] DB changes: SQLite + MySQL + PostgreSQL considered
- [ ] JSON via `common.Marshal/Unmarshal*` (not raw `encoding/json` calls)
- [ ] Frontend: i18n keys via **i18n-translate** skill; `bun run typecheck` if TS changed

### Verification

- [ ] Scoped `go test` for touched packages
- [ ] Frontend typecheck/lint if applicable
- [ ] Manual smoke for user-visible behavior
- [ ] **Pre-commit Codex gate (C2-pre):** companion/`codex review` (or user `/codex:review`) on **combined** base→final tree; mixed committed+dirty → materialize then one `--base origin/main`; findings fixed; re-reviewed until clean **or** explicit user skip/override recorded
- [ ] If Codex unavailable: user chose retry / skip — not silent pass; docs/skill not auto-skipped

### Fork governance

- [ ] Permanent delta? `docs/fork/diff-inventory.md` updated (or explicitly N/A)
- [ ] Inventory risk level honest (`high` if billing/auth/relay core)
- [ ] Regression points written so next sync can re-verify

### Ship

- [ ] `git push -u origin HEAD` (branch only)
- [ ] `gh` targets **AkumaRealLabs/ggapi** (`gh repo view` / `gh repo set-default`)
- [ ] PR base = `main` on team repo (not QuantumNous/new-api)
- [ ] PR describes what / why / how tested
- [ ] No `git push upstream`
- [ ] If UI strings added: **i18n-translate** for all locales including **zh-TW**; sync report clean
- [ ] If CI/workflows: still `runs-on.group: org-linux` unless intentional change + inventory update (GG-003)
- [ ] If update-check: server proxy + PAT write-only (GG-004); no PAT in frontend

---

## §Pre-PR (upstream sync branch)

- [ ] Branch name `sync/upstream-YYYYMMDD`
- [ ] No feature/fix commits mixed in
- [ ] `git fetch upstream` done; PR cites upstream short SHA
- [ ] Conflicts resolved per inventory + Mode E (no whole-tree ours/theirs)
- [ ] `AGENTS.md` still contains ggapi 二开 section if it was present
- [ ] Inventory meta: baseline commit + date updated
- [ ] `needs-rebase` rows restored or still marked with reason
- [ ] Tests: `common` `service` `model` `relay` (and frontend if needed)
- [ ] Smoke: login/token, chat+billing, admin basic page, inventory `active` items
- [ ] Push to **origin** only; PR title `sync: merge upstream/main @ <sha>`

---

## §Post-sync (after sync PR merges)

- [ ] Local `main` pulled
- [ ] Sync remote branch deleted
- [ ] Inventory baseline matches what actually landed
- [ ] Any emergency fix during sync registered as GG-xxx
- [ ] Team notified if `high` behavior changed
- [ ] Optional: note next sync window

---

## §Release

- [ ] DB backup completed
- [ ] Version string strategy (`x.y.z-ggapi.N`) recorded
- [ ] Migrations reviewed for target DB
- [ ] Build: `make build-web` (and image/compose as used in prod)
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
- [ ] Pre-commit Codex gate (C2-pre) still documented if ship-path changed
- [ ] User shown summary; commit only if they asked

### Skill PR

- [ ] Branch `docs/ggapi-fork-…` from latest main
- [ ] PR describes level + why + what not changed
- [ ] GG-002 inventory note still accurate (update if role changed)
