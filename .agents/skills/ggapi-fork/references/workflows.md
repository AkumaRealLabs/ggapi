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
make dev-api    # API + docker dev stack
make dev-web    # default frontend
```

6. Point the user at `docs/fork/README.md` and Mode B for first feature work.

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
4. Frontend `web/default`: Bun, `t('English key')`, hand off i18n to **i18n-translate** skill.  
5. If permanent delta: draft inventory row mentally (ID, path, risk, regression).

### B3. Local verification (scoped)

Backend (only packages you touched):

```bash
go test ./service/...
go test ./model/...
go test ./relay/...
# or narrower: go test ./path/to/pkg -count=1
```

Frontend (if UI/TS touched):

```bash
cd web/default && bun run typecheck
# lint if project scripts require it for the change
```

Manual smoke when behavior is user-visible: login/token, one chat path, admin page.

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

### C2. Commit (only if user asked)

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

3. Build/test.  
4. Restore inventory rows to `active` (or update summary).  
5. Update baseline SHA/date.

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
3. Version suggestion: `x.y.z-ggapi.N` aligned with inventory baseline upstream version.  
4. Minimum regression:

- Login + API token  
- Main inference path + billing  
- Top-up/quota changes if customized  
- All inventory `active` regression points  
- Watch logs for quota saturation / auth errors  

```bash
make build-web
# image/compose per repo Dockerfile / docker-compose.yml
```

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
8. **对齐清单：** 升级后核对 GG-002 摘要与 `skill_version`；提及的永久差异（如 GG-003/004）须与 inventory 一致。
