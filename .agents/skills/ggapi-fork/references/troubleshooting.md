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
| "Should I push or PR?" confusion | §3 |
| Push rejected / non-fast-forward | §4 |
| Merge conflicts during sync | §5 |
| Lost fork customization after sync | §6 |
| Diverged badly from upstream | §7 |
| Dirty tree blocks branch switch | §8 |
| `gh` / PR creation fails | §9 |
| Tests fail after sync | §10 |
| Billing / quota weirdness after custom patch | §11 |
| Afraid of breaking protected branding | §12 |
| Inventory out of date / unknown baseline | §13 |
| Working on wrong remote or fork | §14 |
| Skill keeps rewriting itself / user fears chaotic upgrades | §15 |
| Skill out of date vs docs/fork | §16 |

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

Then Mode C: push branch + PR.

**If already pushed commits to `origin/main`**  
Do **not** force-push `main` unless the user explicitly owns that decision and
team policy allows. Prefer revert PR or forward-fix PR.

---

## §3 Push vs PR (stuck in decision)

Teach this one-liner:

> Push the **branch** so GitHub has your commits; open a **PR** so they can enter **main**.

Never present "only push" or "only PR" as complete landing paths.

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

**Rejected push to main**  
Good — main is protected. Move work to a branch (§2) and PR.

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
3. Still use `sync/upstream-YYYYMMDD` + merge + PR  
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

## §9 Cannot open PR

| Cause | Fix |
|-------|-----|
| Branch not pushed | `git push -u origin HEAD` |
| `gh` not auth | `gh auth status`; user logs in |
| Wrong base | `--base main` |
| No commits vs main | Ensure commits exist: `git log origin/main..HEAD` |
| Already has PR | `gh pr view` / use existing |

Manual fallback: GitHub UI compare `main`...`<branch>` after push.

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
3. Recovery: `git log -- .agents/skills/ggapi-fork` / revert the skill-only PR.  
4. Remind gates: no auto-commit; L2/L3 need plan; Hard rules frozen without explicit L3.  
5. If agent upgraded mid-feature: separate skill commit or revert skill hunks from the feature PR.

## §16 Skill teaches steps that contradict docs/fork

1. **Trust docs** for what you do now.  
2. Diff the relevant section vs skill Mode.  
3. Mode I **L1**: patch skill to match docs (do not “fix” docs unless SOP change is intended).  
4. Bump version + CHANGELOG; ship via docs branch when user wants it on main.

---

## Emergency "I already ran a dangerous command"

| Already did | Mitigation |
|-------------|------------|
| `reset --hard` lost commits | `git reflog` → recover SHA → branch from it |
| Force-pushed topic branch | Coordinate with anyone who pulled; avoid force on `main` |
| Pushed secrets | Rotate credentials; history purge only with team process |
| Merged bad sync to main | Fix-forward PR; inventory note; avoid another hard reset on shared main |
| Skill auto-edited and confused you | See §15; discard uncommitted skill diffs or revert commit |

Always show reflog recovery options before declaring data lost:

```bash
git reflog | head -30
```
