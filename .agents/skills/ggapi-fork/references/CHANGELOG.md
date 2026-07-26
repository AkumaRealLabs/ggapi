# ggapi-fork skill changelog

格式：`## version — YYYY-MM-DD` + 级别 + 摘要。最新在上。

## 1.9.0 — 2026-07-26

- **级别:** L2 / minor（用户指示：C2-pre 审查改为各环境自家 reviewer；**未**改 Hard rules）
- **原因:** 原设计里 Grok 走 Codex companion、Claude/generic 一律回退 Codex CLI；用户要求 Codex 用 Codex 的 review、Grok 用 Grok 的、Claude Code 用 Claude 的
- **变更:**
  - `selectReviewStrategy` 按 surface 路由自家 reviewer：Codex→`codex review --base`；Claude→`claude -p "/code-review origin/main"`；Grok→`grok -p '<审查提示词>'` headless（Grok 无专用 review 子命令）；surface CLI 缺失或 generic 时回退可用的 Codex CLI
  - 移除 Grok Codex companion 动态发现路径（`findCodexCompanion`）
  - 策略新增 `reviewer` 字段；SessionStart 指引与 deny 提示中的 `gate-record --reviewer` 由硬编码 `codex` 改为当前策略 reviewer；fallback 实际执行时记 `codex`
  - `formatReviewCommand` 的 base 替换改为参数内子串替换，支持提示词内嵌 `origin/main`
  - 测试更新：claude-native / grok-native / 缺 CLI 回退 / base 子串替换；删除 companion 用例
  - SKILL.md / workflows C2-pre / troubleshooting §21 / checklists / `docs/fork` README+SOP / GG-002 对齐；`skill_version` → **1.9.0**
- **门禁加固（本轮 C2-pre 审查在 1.8.0 `review-gate.mjs` 中发现，均已修 + 回归测试）:**
  - 裸 `&` 未作分隔符：`git commit -m '<temp快照>' & git push origin HEAD:refs/heads/main` 曾被判为单条合法快照而放行；`&` 现按分隔符处理并排除 `&>` / `2>&1`
  - 分段命中动作后跳过嵌套扫描：`git commit -m '<temp快照>' $(git push origin main)` 同样曾放行；嵌套扫描改为始终执行，命中即把已识别动作标 `compound` → deny
  - 嵌套 shell 仅认 `bash|sh|zsh` + 精确 `-c`/`-lc`：`bash -ec`、`dash -c` 曾零动作放行；改为 7 种 shell + `-[A-Za-z]*c`，读不到脚本且提及 git/gh 时 fail-closed
  - argv 数组载荷 fail-open；改为按元素 shell 引用后拼接（直接 join 会让 `-lc` 只剩裸 token `git` 而再次放行），异常形状记 unresolved；`write_stdin` 纳入 shell 工具
  - 隐式目标 refspec：`git push origin main` 未被识别为推向 `origin/main`；缺省目标现取源 ref 名
  - `git commit` 落的是索引而非工作树：暂存后从工作树删除的内容曾可绕过；提交路径在工作树门禁之外增加索引树比对
  - JS 载荷中 `{cmd: "git" + " push …"}` 拼接被当作已解析常量；仅当字面量后紧跟 `,`/`}` 才视为解析成功
  - `gh release create|delete|edit` 与 `gh api -X POST|PATCH|DELETE …/git/refs` 可绕开 tag 规则创建远端 ref；纳入 tag 分类与「只钉 `origin/main` tip」约束
  - marker 排除不再依赖 `.gitignore`（缺该条目的分支上 `gate-record` 曾永久抛 residue 死锁），改用 `git rm --cached --ignore-unmatch`
  - `.claude/settings.json` hook 路径 `${CLAUDE_PROJECT_DIR}` 对复用该文件的 Grok 不可移植（变量未设 → node 退出 → 无 deny）；改为 `${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel)}`
  - `javascriptToolCalls` 在 `matchAll` 上赋值 `lastIndex` 属空操作，改用 `exec` 循环
  - `shellTokens` 按引号分片切词：`git ""commit`、`git c"o"mmit`、`git 'pu'sh` 均绕过子命令比较；重写为按 shell 单词切分（相邻分片合并为一词）
  - `tokensDescribeShipMutation`（分组/命令替换内的唯一探测器）漏了 `gh release` 与 `gh api …/git/refs`：`(gh release create v9.9.9)` 曾放行；两处判定补齐并加注保持同步
  - shell 侧 mentions 兜底漏 REST merge 与 release：`EP=…/pulls/12/merge; gh api -X PUT $EP` 曾放行
  - `javascriptProperty` 取首个同名键，而 JS 语义取最后一个：`{cmd:"echo ok", cmd:"git push origin main"}` 曾按 `echo ok` 放行；出现重复键即判 unresolved
  - `tagTarget` 返回 null 时 `targetCommit` 默认成 `origin/main`，使「只钉 tip」检查真空通过；改为无法解析即拒绝
  - `PreToolUse` 收到 shell 工具但 `tool_input` 缺失时静默放行，与模块自身「读不懂不得放行」原则相悖；改为拒绝
  - 新增 `make test-agent-skill`：本套安全关键回归此前无任何 runner（makefile 与 CI 均未跑），可静默回归
  - `docs/fork` SOP 更正 Hook 能力边界：明确 `git merge`/`cherry-pick`/`revert`/`am` 与 `push --force` **不在**机械拦截范围
- **跨厂商回退移除（用户纠正）:** 初版给 Claude/Grok 策略挂了 `|| codex review` 回退，实战中 `claude -p` 撞限额即回落到 Codex 审查——这恰好抵消了「各环境用自家 reviewer」的本意。现在：Claude→`claude -p "/code-review <base>"`（无 `claude` CLI 时为 `claude-in-session`，提示在会话内跑 `/code-review`），Grok→`grok -p`，二者均**无**回退；**只有** generic（本身没有 reviewer）才用 Codex CLI。自家 reviewer 装了但失败（限额/配额/鉴权）= **gate 阻塞**，按 §21 停，**禁止**换另一家 reviewer 顶上
- **零退出码不等于审查跑过:** `claude -p` 撞 session 限额时打印提示并以 **0** 退出（因此任何 `||` 链都不会触发）。必须读输出：无 findings **且**无审查正文 = 阻塞，不是干净，不得 `gate-record`。已写入 §21 与 workflows
- **第 4 轮审查修复:**
  - 索引比对被 `state !== "empty"` 关掉：工作树树等于 base 时（gate 状态 empty），`git add x && rm x && git commit` 仍能落未审内容——而索引恰是此时唯一的载体；去掉该条件
  - `commandTokens` 剥掉前导 `VAR=value`，使 `GIT_DIR=` / `GIT_WORK_TREE=` / `GIT_INDEX_FILE=` 等重定向环境永远到不了 `gitInvocationViolation`，绕过「不得覆盖 Git 配置或仓库路径」；新增 `leadingAssignments` 并回接到 `overridesRepository`
  - `git push --tags` / `--follow-tags` 批量发布本地 tag，而只有分支 refspec 被检查，绕过「tag 只钉 tip」；要求显式 tag refspec
  - `event !== "PreToolUse"` 直接 return：事件名缺失或拼写不同的载荷零输出＝放行；改为只要携带 tool 调用就照常校验
  - JS 侧 `mentionsShipMutation` 漏 `gh release` / `git/refs`（shell 侧已补），读不懂的 `exec` 载荷建 release 曾放行
  - **回归（本轮引入）:** `tagTarget` 对删除返回 null，叠加上一轮「无法解析即拒绝」后把 SOP 记载的本地 `git tag -d` 清理一并封死；本地删除不触及远端 ref，恢复放行，远端删除仍走 push/gh 门禁
  - **回归（本轮引入）:** 移除跨厂商回退后未同步文档，README/SOP/SKILL 仍写「缺 CLI 回退 codex」——照文档操作恰好执行 §21 禁止的跨厂商替代；三处已改，并把 Grok 改为与 Claude 对称的 `grok-in-session`
  - `.claude/settings.json` 转为跟踪会覆盖开发者本地未跟踪的同名配置；SOP 增加升级警示与 `settings.local.json` 建议
- **第 5 轮审查修复:**
  - `optionValue` 不认粘连短选项：`gh api -XPUT …/pulls/12/merge`、`-XDELETE …/git/refs/…` 解析成 method GET，未审 merge 与远端删 tag 直接放行
  - `git push origin --delete refs/tags/v1.0.0` 走 `{deletion:true}` 自动放行，与 `gh` 侧对远端 tag 的拦截不一致；改为区分远端 tag 删除（拒绝）与分支删除（放行）
  - `--dry-run` / `--help` / `-h` / `-n` 对所有 token 匹配，包括**选项的值**：`git commit -m --dry-run` 是真实提交却被判为空操作；新增 `hasFlag` 跳过取值选项的实参
  - `GIT_REDIRECTING_ENVIRONMENT` 漏 `GIT_CONFIG_COUNT` / `GIT_CONFIG_KEY_n` / `GIT_CONFIG_VALUE_n` / `GIT_CONFIG_PARAMETERS`，等价于被明确拦截的 `git -c`（可改写 `remote.origin.url`）
  - **回归（本轮引入）:** 索引比对用 `tokens.includes("-a")`，漏掉捆绑写法 `-am`，导致 `git commit -am` 被误判「索引与已审树不符」
  - `bash --version` / `sh --help` 被误判为持久 shell 会话而拒绝
  - 测试 fixture 未 `realpathSync`，macOS 上 `os.tmpdir()` 是符号链接，`gitInvocationViolation` 会拒绝全部动作，`make test-agent-skill` 在 macOS 必然失败
  - **回归（本轮引入）:** §21 反模式表仍写「回退 `codex review` CLI」，与其上方刚确立的禁止跨厂商替代自相矛盾
- **第 6 轮审查修复:**
  - **回归（上轮引入）:** 远端 tag 删除守卫只匹配 `refs/tags/…`，裸名 `git push origin --delete v1.0.0-rc.21.1` 仍自动放行，且我为此写的 `!ref.includes("/")` 判断是死代码；改为向 git 查询该名是否解析为 tag
  - **回归（上轮引入）:** 新加的 `stagesWorktree` / `deletesLocalTag` / `tagTarget` 未使用同轮引入的 `hasFlag`：`git commit -m -a`（`-a` 是**消息内容**）会跳过索引比对，`git tag -m -d v9` 被误判为本地删除
  - `containsNestedShipMutation` 只看命令替换体内的首个词，而引号会使整体不被切段：`git commit -m "$(true; git push origin HEAD:refs/heads/main)"` 判为普通非复合提交并放行；新增 `commandSubstitutions` 解析 `$( )` / 反引号正文
  - `codex` surface 缺 `codex` 二进制时返回 `unavailable`（§21 硬阻塞），与 Claude/Grok 的 `*-in-session` 不对称；补 `codex-in-session`
- **第 7 轮审查修复:**
  - `gh api` 的动词算错：传了 `-f/-F/--field/--input` 时 gh 自动改用 POST，而分类器仍默认 GET，`gh api …/git/refs -f ref=refs/tags/v9.9.9 -f sha=<oid>` 可无门禁创建远端 tag；新增 `ghApiMethod` 统一计算，五处调用点改用
  - Hook 只按会话 `cwd` 解析仓库，忽略工具调用自带的 `workdir`（Codex `shell.workdir`）：指定另一个 worktree 时，会拿本仓的干净记录去放行那边的未审 tip；改为优先解析调用自身的工作目录（`git -C` 早已有等价守卫）
  - 「禁止直推 main」只匹配字面 `main` / `refs/heads/main`，漏 `heads/main`（git 同样解析为 `refs/heads/main`）；抽出 `targetsMainBranch`
  - **回归（上轮引入）:** 远端 tag 删除判断里 `if (ref.includes("/")) return false;` 在查 tag 之前短路，含斜杠的 tag 名（如 `release/2026-07`）被当作分支删除放行；tag 名允许含斜杠，不能按形状跳过查询
  - 门禁测试无 CI runner 一项，由寄存中的 `ci.yml`（`agent-skill` job 跑 `make test-agent-skill`）解决
- **第 8 轮审查修复（三项均为前几轮自身留下的口子）:**
  - `javascriptProperty` 的常量分支缺少字面量分支已有的「其后须为 `,` 或 `}`」守卫：`const C = "echo ok"; tools.exec_command({cmd: C + "; git push origin main"})` 解析成 `echo ok` 并放行；字面量等价写法本已拦截
  - `pushedTree` 只按字面 `refs/tags/` 前缀识别 tag 推送，而 `--delete` 路径已改为解析裸名：`git tag v9.9.9 && git push origin v9.9.9` 被当作分支推送，可在非 tip 处创建远端 release tag；抽出 `resolvesToTag` 供两处共用
  - **回归（上轮引入）:** `hookRoot` 对每个载荷都急切解析 `workdir` 并交给 `findRepositoryRoot`，路径不在仓库内即抛错，被 catch 转成硬 deny——任何带仓外 `workdir` 的**只读**命令都会被拒；改为解析失败时回落到会话仓库
- **第 9 轮审查修复（五项均为 fail-open）:**
  - 非 shell 解释器参数内的发车动作不匹配任何分支：`node -e "…execSync('git push origin HEAD')"`、`python3 -c` 等直接放行；新增 `INLINE_INTERPRETERS`，`-e`/`-c` 载荷提及 git/gh 即判 uninspectable
  - 嵌套 shell 分支在 `leadingAssignments` 之前 `continue`：`GIT_CONFIG_COUNT=1 … sh -c 'git push origin HEAD'` 绕过环境重定向守卫，而未包装的等价写法被正确拒绝
  - `resolvesToTag` 只查本地 tag，按 SOP §5 先做本地清理后，`git push origin --delete v1.0.0` 会被当作普通分支删除，无门禁删掉已发布的 release tag。**注意：** 我最初改成「非已知分支即拒」，结果把「本地不存在的分支」也误伤（3 个既有测试变红）；最终改为**要求消歧**——裸名若不是已知分支，提示改写为 `refs/heads/…` 或 `refs/tags/…`
  - `gh api …/releases -f tag_name=…`（创建远端 tag）与 `gh api graphql -f query='mutation { mergePullRequest… }'` 两者都不匹配既有探测器，且兜底正则的 `\brelease\b` 匹配不到复数 `releases`；新增 `GH_API_RELEASE_SURFACE`，graphql 一律判 uninspectable
  - `SHELL_TOOLS` 是固定白名单，harness 若把 exec 工具命名为别的（如 `local_shell`），门禁**静默零生效而外观仍像装好了**。此项无法靠枚举根治，改为由 `detect` 输出 `enforcedShellTools` 显式暴露前提，接入新环境时须核对
- **定位澄清（重要）:** Hook 属**尽力拦截 / 纵深防御**，不是不可绕过的安全边界。其检测为黑名单式（枚举已知命令写法），原理上无法穷尽；五轮审查共修 39 处、含 8 处可利用绕过，且每轮仍有新发现——这是该设计的固有性质，不是尚未修完。文档已相应下调表述。若要当硬边界用，须改为白名单式（只放行少量完全限定的规范命令，未知写法按构造 fail-closed），而非继续给黑名单打补丁
- **已知未修（须知悉）:** `push --force` 不被机械拦截（属 Hard rules「先问用户」，机械拒绝会与用户授权的合法强推死锁）；只读命令若在字符串字面量中含发车词（如 `node -e '… "git push origin main" …'`）会被误拒（fail-closed，改用脚本文件规避）
- **未改:** Hard rules；gate 记录/失效机制、材料化流程、禁自跳与用户 opt-out 语义均不变

## 1.8.0 — 2026-07-26

- **级别:** L2 / minor（当前 Agent 原生适配与项目 Hook；**未**改 Hard rules）
- **原因:** 工作流仍把 Grok Codex companion 当默认入口；用户要求自动识别 Codex/Grok/Claude，并在 Codex 中使用原生 Hook
- **变更:**
  - 新增 `scripts/agent-adapter.mjs`：环境/能力探测；Codex 原生 review、Grok companion 动态发现、Claude/generic CLI fallback
  - 新增 `.codex/hooks.json` 与 `.claude/settings.json`：`SessionStart` 运行时指引 + `PreToolUse` 发车门禁；Grok 通过其 Claude Hook 兼容发现复用同一入口，避免重复 Hook
  - `.gitignore` 继续忽略 `.claude` 个人状态，仅放行项目级 `settings.json`
  - gate marker 写入 Git 忽略的 `.ggapi-agent/`（workspace sandbox 可写）：临时 index 计算完整最终树（tracked + untracked、排除 ignored）；同树 commit/amend 有效，内容/分支/`origin/main` 变化自动 stale
  - clean review 用 `gate-record`；仅明确用户 opt-out 可用带原因的 `gate-bypass`；Hook 不自动 commit/push/merge/tag，不写机器 trust hash
  - C2-pre 统一为 dirty final tree 先 materialize、当前 surface 一次 `--base origin/main` review；保留固定 temp snapshot 消息的窄放行
  - 发车动作必须是可静态检查的单条命令；支持 `git -C/-c`，拒绝复合、嵌套 shell、动态拼装、truncated payload 绕过；Codex 后续 `write_stdin` 不重跑 `PreToolUse`，因此拒绝裸持久 shell；按实际 commit/ref/tag/PR head tree 校验 marker，push 显式限定 `origin`，merge 必须带 reviewed head pin
  - clean `gate-record` 要求 materialized review 后工作树无 tracked/untracked 残留，防止把 `codex review --base` 未覆盖的内容错误标绿
  - release SOP 先打印、再使用实际字面量执行独立 `git tag` / `git push`，避免变量与 `||` 错误分支被 Hook 误判
  - Grok companion 失败时由生成命令实际回退 `codex review`；新增 Node 回归测试覆盖 runtime 路由、命令分类、clean/bypass/stale、同树 commit、base 漂移、实际推送/合并对象、子目录启动及三种 deny 协议
  - 文档/SOP/checklists/troubleshooting/GG-002 对齐；`skill_version` → **1.8.0**
- **未改:** Hard rules；不自动 commit/push/merge/tag；C2-pre 仍可由**用户**明确 opt-out；项目 Hook 仍需每台机器显式 trust

## 1.7.0 — 2026-07-17

- **级别:** L2 / minor（Mode C 续跑 + C2-pre 强化；**未**改 Hard rules）
- **原因:** PR #26 全链路：`继续` 在 PR 已开时被当成再写代码；用户拒绝 agent 侧跳过 Codex；gate 须按树内容失效而非仅 SHA
- **变更:**
  - Mode **C-continue**：按阶段路由 `继续`（不默认 Mode B；不补 push/PR 词；C0 已含 merge 则续 C5；**仅**本单元跑过 C2-pre 时做 stale 检查；push-only 链不发明 gate）
  - **C2-pre**：禁 agent 自跳；修完再审；fingerprint=`write-tree`（含 untracked）；C2/C3 后更新 **merge pin**；C5 比对 remote `headRefOid`；`origin/main` SHA 漂移才 stale（非 count>0）；companion→CLI
  - checklists / troubleshooting §21 对齐；GG-002 → **v1.7.0**；`docs/fork/README` skill 版本锚点
  - `skill_version` → **1.7.0**（L2 走 minor，不用 1.6.1 patch）
- **未改:** Hard rules；不自动 commit/push/merge；C2-pre 仍可由**用户**明确 opt-out

## 1.6.0 — 2026-07-16

- **级别:** L2 / minor（Mode C/F 发车实战补齐；**未**改 Hard rules）
- **原因:** PR #23 全链路：oxlint 偏晚、`gh pr merge` GraphQL EOF 误判失败、tag 后未等 GHCR、一句话「提交→…→发版」缺链式语义
- **变更:**
  - Mode **B3** / **C1b** / Pre-PR：typecheck（ggapi/default）+ **路径级** oxlint/eslint（不全树 `bun run lint` 当硬门禁；基线有债）
  - Mode **C0**：链式发车（缺词不补权；**发版≠merge**；push/PR 不推断 commit）
  - Mode **C5** + **§23**：查 state/autoMerge；EOF ≠ 失败；**MERGED 前不发版**
  - Mode **F** + **§24**：`gh run list --branch $REL_TAG` + `watch --exit-status` 再称发版完成
  - classic：无 typecheck；勿套用 ggapi 命令
  - `skill_version` → **1.6.0**；GG-002 摘要对齐
- **未改:** Hard rules；不自动 commit/push/merge；不自动打 tag

## 1.5.1 — 2026-07-12

- **级别:** L1 / patch（Mode F 对齐 docs：默认 tag→GHCR；**未**改 Hard rules）
- **原因:** 用户要 tag 推 GHCR，且默认不构建裸二进制
- **变更:**
  - Mode F / Release checklist：默认产物 **GHCR**；`release.yml` 仅手动；禁 Docker Hub
  - 锚点 GG-003 文案；`skill_version` → **1.5.1**
- **未改:** Hard rules；不自动 commit/push

## 1.5.0 — 2026-07-12

- **级别:** L2 / minor（Mode F 发版 tag 流程 + SOP §5.1；**未**改 Hard rules）
- **原因:** 用户确认本仓 git tag 采用 `v<上游基线>.N`（例 `v1.0.0-rc.20.1`），不再推荐 `x.y.z-ggapi.N`
- **变更:**
  - `docs/fork/branch-and-sync-sop.md` 新增 **§5.1**
  - 清单基线 **`v1.0.0-rc.20`**；GG-003 Docker 防护（后续 1.5.1 / GHCR PR 再演进）
  - Mode F / Release checklist 与 SOP 对齐
  - `skill_version` → **1.5.0**
- **未改:** Hard rules；不自动 commit/push；不自动打 tag

## 1.4.0 — 2026-07-12

- **级别:** L2 / minor（第三壳教练 + #7 发车实战；**未**改 Hard rules 表与 origin/main/upstream/品牌/计费语义）
- **原因:** 用户确认 Mode I 方案；`docs/fork` 已有 GG-005 / SOP §3.5，skill 仍教 `web/default`；吸收 PR #7 审查与合入教训
- **变更:**
  - SKILL：`skill_version` → **1.4.0**；inventory 锚点 **GG-005**；Product shell 表；Mode I 触发词 **自提升**；description 含第三壳
  - Mode A/B：默认本地栈与 typecheck → **`web/ggapi`**；功能追 default 的 port 提示
  - Mode C：**C1b** 三壳/embed/品牌/i18n 检查（**置于 C2-pre 之前**；改后须再审）；materialize 临时 commit 禁止推送 temp 信息
  - Mode D/E/F：sync 不夹带 ggapi 大皮肤；post-sync port follow-up；release **`make build-all-web`**（本地 go embed 三壳齐全）
  - Mode B：typecheck 用 subshell，`make` 始终在仓库根；按**实际编辑的壳**选 `build-web*`
  - checklists：第三壳 Pre-PR 段；双壳 theme 选择器；post-sync / release 勾选项
  - troubleshooting **§22** 第三壳/主题/embed（default+ggapi 管理端均需 ggapi 选项）；§20 补 orphan keys + ggapi 路径
  - `docs/fork/branch-and-sync-sop.md` 发版/命令速查对齐 `make build-all-web` 与产品壳 dev
  - 依赖 skill：**i18n-translate** / **shadcn-ui** shell-aware（每条命令写全路径 `web/ggapi` 或 `web/default`）；i18n 含 **zh-TW**；classic 走 `i18next-cli`
  - C1b / §22：classic `frontendTheme` 切换路径纳入 theme 门禁
  - `web/{ggapi,default}/scripts/sync-i18n.mjs`：`zh-TW` 纳入 untranslated 非拉丁判定；字面量 allowlist 补 `Webhook`/`Gotify`
  - GG-002 清单摘要对齐 v1.4.0
- **未改:** Hard rules；不自动 commit/push/merge；C2-pre 仍可用户明确 opt-out

## 1.3.0 — 2026-07-11

- **级别:** L2 / minor（发车流程 / Ship quality gate；**未**改 Hard rules 表与 origin/main/upstream/品牌/计费语义 → 非 L3；self-upgrade §2 + §6 已对齐「质量门禁≠安全语义 major」）
- **原因:** 用户要求每次最终提交前跑 `/codex:review`，有问题则修复后再审
- **变更:**
  - Mode C 新增 **C2-pre** + SKILL「Ship quality gate」（审查环，**不在** Hard rules 表内）
  - 触发：`提交`/`commit` 进入 Mode C；agent 用 **companion CLI**（不依赖 slash）
  - 范围：**单次** base→最终树审查；混合「已提交 + 脏工作区」时先 **materialize** 再 `--base origin/main`
  - 文档/skill **不可**单方面跳过（须用户明确 opt-out）
  - checklists / troubleshooting **§21**；`docs/fork` README+SOP；GG-002 → v1.3.0
  - `skill_version` → 1.3.0
- **未改:** Hard rules 表（origin-only / 禁直推 main / 品牌 / 计费 / 自升级红线）；不自动 commit/push；审查本身仍为 review-only

## 1.2.0 — 2026-07-11

- **级别:** L2（含 L1 实战条目）
- **原因:** 用户确认 Mode I 方案；吸收本会话已合入的 GG-003/GG-004 与发车踩坑
- **变更:**
  - Mode C：`gh` 必须指向 `AkumaRealLabs/ggapi`；提供 `gh api repos/…/pulls` 回退；合并清理标准命令
  - troubleshooting §17–§20：`gh` 错仓、org-linux CI、私有仓检查更新 PAT、i18n zh-TW
  - checklists：目标仓 / zh-TW / GG-003 / GG-004 勾选项
  - SKILL：触发词与 inventory 锚点 GG-001–004；`skill_version` → 1.2.0
- **未改:** Hard rules、中文教练默认、不自动 commit/push

## 1.1.0 — 2026-07-11

- **级别:** L2
- **原因:** 用户要求 skill「能自主升级、但不乱升级」
- **变更:**
  - 新增 Mode **I**（skill 自升级）与触发词
  - 新增 `references/self-upgrade.md`（分级 L0–L3、闸门、禁止项、与 docs 优先级）
  - `SKILL.md` 增加自升级硬规则与 `metadata.skill_version`
  - 本 CHANGELOG 启用
- **未改:** origin/main/upstream 硬规则、中文教练默认、commit/push 仍须用户明确授权

## 1.0.0 — 2026-07-11

- **级别:** 初始
- **变更:** 首版 playbook（Modes A–H、workflows / troubleshooting / checklists、中文教练）
