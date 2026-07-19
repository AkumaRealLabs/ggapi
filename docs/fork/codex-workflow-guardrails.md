# ggapi 跨运行面工作流守卫

本仓库把日常二开和发车拆成四层：

```text
AGENTS.md / docs / ggapi-fork skill
        ↓ 统一阶段、授权和审查范围
Codex 项目 Hooks
        ↓ 开始、过程、结束上下文（仅 Codex App / CLI）
execpolicy + Git Hooks
        ↓ 确定性危险操作门禁（所有客户端和人工终端）
GitHub Actions PR CI
        ↓ pull_request 代码质量检查
```

## 1. 适用范围

`ggapi-fork`、仓库 Git Hooks 和 PR CI 共同覆盖 Codex App、Codex CLI、Grok
Build CLI 以及人工终端。`.codex/hooks.json` 和 `.codex/hooks/` 是 Codex
App/CLI 的增强层，Grok 不依赖它们，仍通过 `$ggapi-fork` / `/ggapi-fork` 和
Git Hooks 执行相同流程。

项目 Hooks 只在仓库被信任后加载。首次打开仓库后，在 Codex 中运行
`/hooks`，逐项检查脚本路径和 hash，确认准确后再信任；hash 变化会重新要求
审核。Hook 输出是上下文，不是授权凭证，也不是可靠的硬阻断。

## 2. 阶段与授权

标准阶段固定为：

```text
B → C1 → C2-pre → C2 → C3 → C4 → C5 → F
```

| 阶段 | 含义 | 必要授权 |
|------|------|----------|
| B | 任务确认、开 topic 分支 | 修改代码不等于 ship 授权 |
| C1 | 实现和本地验证 | 不自动 commit |
| C2-pre | 当前最终树的 Codex 审查 | 审查不等于 commit |
| C2 | 创建 commit | 当前消息明确写 `提交` / `commit` |
| C3 | 推送 topic 分支到 `origin` | 当前消息明确写 `push` / `推送` |
| C4 | 创建或更新 PR | 当前消息明确写 `PR` / `开 PR` |
| C5 | 合并 PR、清理分支 | 当前消息明确写 `merge` / `合并` |
| F | tag/release/部署回归 | 当前消息明确写 `tag` / `发版` / `部署` |

授权规则：

1. 只读取**用户当前消息**中的明确动词；一次消息写出完整链条时，按出现顺序执行。
2. 授权不持久化。历史消息、dirty tree、已有 commit、PR 状态和“继续”都不能补推新的 commit、push、PR、merge 或 release 权限。
3. `开 PR` 不隐含 push；分支尚未推送时停在 C3，等待明确 push。单独 `合并` 不隐含 PR 创建；没有 PR 时停在 C4。`开 PR 并合并` 明确同时授权 C4/C5，可在已推送分支上先创建缺失的 PR 再合并。通过 CI 不隐含 merge。
4. execpolicy 的 `prompt` 只是确认弹窗，不是用户业务授权；Git Hook 的允许也不是授权。两者都不能替代当前消息中的动词。
5. 没有 PR 时必须报告原因，例如 `PR: 未创建，等待 commit + push`，不能省略字段。

## 3. Ship Status Contract

Codex App/CLI 的 `Stop` Hook、skill、PR 模板以及所有运行面的最终回复都使用
以下字段。字段必须全部出现，值以实际命令和远端结果为准：

```text
当前阶段:
当前分支:
Commit:
Push:
PR:
Merge:
diff-inventory:
验证:
下一步授权:
```

Hook 不读取或深度解析 `transcript_path`，不把 `gh` 不可用当成 PR 成功，不伪造
测试通过。`Stop` 最多请求一次补充状态块，使用 Codex 提供的
`stop_hook_active` 避免循环继续。

## 4. 各层职责

### Codex Hooks

- `SessionStart`：只用本地 Git 读取 remote、branch、dirty、ahead/behind 和 tracking；不 fetch、不访问 GitHub、不改文件。
- `UserPromptSubmit`：每轮提醒当前消息授权边界；不以正则替代模型意图判断。
- `PreToolUse` / `PostToolUse`：对 commit、push、PR、merge、tag 提供阶段和结果上下文；不声称能可靠阻断。
- `Stop`：仅在本轮已观察到 ship 命令时要求完整状态块；review 动作和未执行 ship 的结构化审查轮次直接放行，避免破坏原生 review 输出；不依赖 transcript 格式。

### execpolicy

`.codex/rules/ggapi.rules` 对 upstream、QuantumNous URL、`origin/main` push、
`--no-verify` push 和常见的 `core.hooksPath` per-command 绕过作 `forbidden`；
commit、其他 push、PR create/merge、tag 和 release create 作 `prompt`。
规则通过 `codex execpolicy check` 验证，并与用户层已有 allow 规则取最严格决策。

### Git Hooks

运行 `make setup-git-hooks` 设置本仓 `core.hooksPath=.githooks`。`pre-commit`
拒绝 `main` 常规提交；`pre-push` 只允许名为 `origin` 且 URL 指向
`AkumaRealLabs/ggapi` 的远端，并拒绝目标 `refs/heads/main`。Git Hooks 不读取聊天记录、
不记录授权、不提供环境变量绕过；execpolicy 另行禁止标准 `--no-verify` 和常见
`git -c core.hooksPath=...` 绕过形式。

### PR CI

`.github/workflows/ci.yml` 使用 `pull_request`（不是 `pull_request_target`），只给
`contents: read`，运行在 `org-linux`，按 PR 编号取消旧运行。先按 base/head SHA
分类，再编译根 executable/router、运行 scoped Go tests、触及壳的 typecheck/改动路径 lint；classic Prettier 忽略不支持的二进制资源；只有依赖、embed、
Dockerfile、makefile、release workflow 或主题接线变更才运行 `make build-all-web`。
所有 job 在未触及范围时也输出明确的 skip 步骤。

`.github/workflows/pr-check.yml` 仍是只读的模板/anti-slop 检查，不 checkout 或执行
PR 代码；事件扩展为 `opened`、`reopened`、`edited`、`synchronize`。

当前 GitHub 私有套餐不能启用 branch protection，因此 CI 暂时是可见门禁，不伪称为
服务端强制 required check。PR head 变化会触发 CI；skill 必须把旧 review 标记为
stale 并按 C2-pre 重新审查。

## 5. 本地验证

```bash
node --test .codex/hooks/ggapi-workflow.test.mjs
codex execpolicy check --pretty --rules .codex/rules/ggapi.rules -- git push upstream main
codex execpolicy check --pretty --rules .codex/rules/ggapi.rules -- git push --no-verify origin main
codex execpolicy check --pretty --rules .codex/rules/ggapi.rules -- git push origin feat/example
make setup-git-hooks
git diff --check
```

Git Hooks 应在临时仓库中验证 `main` commit、upstream/main push 拒绝，以及
origin/topic push 允许。完整 CI 对应本地命令是 scoped Go tests、两个 default/ggapi
typecheck、改动路径 lint 和必要时 `make build-all-web`。
