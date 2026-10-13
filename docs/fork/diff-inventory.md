# ggapi 二开差异清单

用于登记本仓库相对 **upstream（QuantumNous/new-api）** 的长期定制，降低同步冲突成本，并保证发版可回归。

关联流程见 [分支命名与同步 SOP](./branch-and-sync-sop.md)。

---

## 1. 元信息

| 字段 | 填写说明 | 当前值 |
|------|----------|--------|
| 清单维护人 | 负责督促更新的人 | Akuma-real |
| 最近更新日期 | `YYYY-MM-DD` | 2026-10-13 |
| 基准 upstream 版本 | 最近一次完整同步时的 tag 或 `VERSION` | **`v1.0.0-rc.44`**（`633b4c5b1`；本次合入 304 个上游提交，并按“能用上游代码就用上游”收敛二开差异：GG-006 删除，GG-008/009/011/012 改用上游实现，GG-004/007/010 以最小补丁重放在上游代码上。官方前端保持 `web/`。发版 `N` 自 `.1` 起递增） |
| 基准 upstream commit | 最近一次 `merge upstream/main` 的提交 SHA（短哈希即可） | `633b4c5b1` |
| 本仓版本策略 | git tag 格式（见 [SOP §5.1](./branch-and-sync-sop.md)） | **`v<上游基线>.N`**（例已发 `v1.0.0-rc.25.1`；本次新基线发版从 **`v1.0.0-rc.44.1`** 起）；基线须为 `origin/main` 祖先；正式 tag **只钉** `origin/main` tip；**禁止**与 upstream 同名 tag；`N` 仅在「基准 upstream 版本」串变化时归 1；**默认 tag 产物 = GHCR 镜像**（非裸二进制） |

同步完成后，务必更新「基准 upstream 版本 / commit」与「最近更新日期」。

---

## 2. 填写规范

### 2.1 何时登记

满足任一条件即新增或更新一行：

- 合并进 `main` 后，相对 upstream 会**长期保留**的代码/配置/文档差异
- 修改了上游已有文件（即使只有几行）
- 新增了仅本仓存在的包、渠道、脚本、部署清单
- 上游同步时发生冲突并保留了本仓逻辑

### 2.2 何时更新状态

| 动作 | 状态调整 |
|------|----------|
| 定制已合入上游官方并删除本仓补丁 | → `upstreamed` |
| 定制废弃、代码已回退 | → `dropped` |
| 即将/正在同步上游，该项可能冲突 | → `needs-rebase` |
| 冲突已解决且行为仍符合预期 | → 回到 `active`，必要时改「差异摘要」 |

### 2.3 类型（`类型` 列）

| 值 | 含义 |
|----|------|
| `config` | 仅默认配置、环境变量示例、部署参数差异 |
| `feature` | 本仓新功能（优先独立目录/包） |
| `patch` | 对上游已有文件的行为补丁 |
| `branding-safe` | 在**不删除**上游品牌/归因前提下的展示层调整 |
| `infra` | CI、Docker、makefile、内部运维脚本等 |
| `docs` | 仅文档（如本目录） |

### 2.4 上游冲突风险

| 值 | 含义 | 典型范围 |
|----|------|----------|
| `low` | 独立文件/目录，上游很少改动 | `docs/fork/`、自有 `pkg/xxx/`、新 channel 目录 |
| `medium` | 旁路接入或半热点文件 | router 注册点、setting 扩展、前端 feature 入口 |
| `high` | 上游频繁演进或资金/安全相关 | `service/*quota*`、`model/` 事务与锁、`relay/`、鉴权 middleware、`pkg/billingexpr/` |

`high` 项应尽量收敛：能抽到独立文件就不要继续堆在上游热点路径。

### 2.5 状态

| 值 | 含义 |
|----|------|
| `active` | 生产依赖，同步时必须保留或等价重放 |
| `needs-rebase` | 同步中/待验证，暂勿当稳定基线 |
| `upstreamed` | 已由官方吸收，本仓差异应删除 |
| `dropped` | 已废弃 |

### 2.6 禁止事项

- 不得删除、替换或弱化 **new-api**、**QuantumNous** 等受保护项目信息（见 `AGENTS.md`）
- 不得把密钥、生产连接串、客户数据写进清单
- 不得用清单代替 Code Review；`high` 风险项仍须二审

---

## 3. 表格模板（复制用）

```markdown
| ID | 类型 | 路径/范围 | 差异摘要 | 原因 | 上游冲突风险 | 回归要点 | 状态 | 负责人 | 引入日期 | 关联分支/PR |
|----|------|-----------|----------|------|--------------|----------|------|--------|----------|-------------|
| GG-xxx | feature | `pkg/example/` | … | … | low | … | active | … | 2026-07-11 | `feat/…` #… |
```

列说明：

| 列 | 要求 |
|----|------|
| ID | 稳定编号，建议 `GG-xxx`，不要复用已关闭编号 |
| 路径/范围 | 尽量精确到文件或目录；跨多处可写「主路径 + 见 PR」 |
| 差异摘要 | 一句话说清「相对上游多了/改了什么」 |
| 原因 | 业务或技术动机；写清为何不能仅靠配置 |
| 回归要点 | 发版或同步后如何验证该项仍正确 |
| 关联分支/PR | 便于追溯；可写 issue 号 |

---

## 4. 示例（虚构，勿当真实差异）

> 以下仅演示填法，**不是**当前仓库真实定制。正式清单以第 5 节为准。

| ID | 类型 | 路径/范围 | 差异摘要 | 原因 | 上游冲突风险 | 回归要点 | 状态 | 负责人 | 引入日期 | 关联分支/PR |
|----|------|-----------|----------|------|--------------|----------|------|--------|----------|-------------|
| GG-EX01 | docs | `docs/fork/` | 二开 SOP 与差异清单 | 长期维护需要 | low | 文档链接可打开 | active | demo | 2026-07-11 | `docs/fork-sop` |
| GG-EX02 | feature | `relay/channel/acme/` | 新增 Acme 渠道适配 | 客户上游协议 | low | 该渠道 chat 通 + 计费日志正常 | active | demo | 2026-07-01 | `feat/acme-channel` #12 |
| GG-EX03 | patch | `service/quota.go` | 预扣失败时返回扩展错误码 | 与内部网关约定 | high | 余额不足/预扣饱和路径；对照上游 changelog | needs-rebase | demo | 2026-06-15 | `fix/quota-error-code` #8 |

---

## 5. 正式清单

> 仅登记相对 upstream 的**永久**差异。第 4 节虚构示例不要混入本表。新增/变更定制后同步更新元信息。

| ID | 类型 | 路径/范围 | 差异摘要 | 原因 | 上游冲突风险 | 回归要点 | 状态 | 负责人 | 引入日期 | 关联分支/PR |
|----|------|-----------|----------|------|--------------|----------|------|--------|----------|-------------|
| GG-001 | docs | `docs/fork/`；`AGENTS.md`（「ggapi 二开维护」小节） | 新增二开文档索引、差异清单、分支与上游同步 SOP；在 `AGENTS.md` 增加短入口与文档链接 | 长期二开需要可追踪差异与同步流程；工程规范仍以 `AGENTS.md` 全文为准 | medium（`AGENTS.md` 上游亦会改动；`docs/fork/` 为 low） | 打开 `docs/fork/README.md` 与三份文档内链；`AGENTS.md` 二开段落与表格链接可访问；同步冲突时优先保留本仓二开入口再合入上游工程规则 | active | Akuma-real | 2026-07-11 | `docs/fork-sop` |
| GG-002 | docs | `.agents/skills/ggapi-fork/`；`docs/fork/README.md` / `AGENTS.md` 入口链接 | `/ggapi-fork` 教练 skill **v4.0.0**（Modes A–I + 可控自升级 + **本地合并到 main、不走 PR** + **C-continue** + 链式 C→F + 官方单前端 + **tag `v<上游>.N` → GHCR wait**）；含 oxlint 发车、merge flake 校验、GitHub-hosted runner、检查更新 PAT、i18n zh-TW 等 | 二开需要命令级教练，且 skill 须能跟进文档/实战而不静默削弱安全规则 | low（独立 skill 目录；入口链接触及 `AGENTS.md`/`docs/fork` 为 medium 旁路） | `/ggapi-fork` 可发现；前端统一为官方 `web/`，GG-005 已退役；tag→GHCR 见 SOP §5.1 / GG-003；用户说提交后直接 commit，不再停下等待人工终端审查；用户说合并/推送后在本地 `--no-ff` 合入 `main` 并推送 `origin/main`，不开 PR、不强推，推送后等 `main` 的 CI；已提交或已合入时「继续」不重写代码且不补合并词；自升级 L1 可小补、L2/L3 须确认、不自动 commit；文档优先于 skill | active | Akuma-real | 2026-07-11 | `docs/fork-sop` / `docs/ggapi-fork-*` / `docs/version-tag-scheme` / `chore/tag-ghcr-no-binary` / `docs/ggapi-fork-1.6.0` / `docs/ggapi-fork-1.7.0` / `chore/manual-review-hosted-runners` / `sync/upstream-20260727` / `docs/remove-anti-slop-policy` / `sync/upstream-20261013` / `docs/ggapi-fork-no-pr` |
| GG-003 | infra | `.github/workflows/*`（`ci.yml`、`docker-build.yml`、`docker-image-branch.yml`、`release.yml`、`electron-build.yml`）；`.github/PULL_REQUEST_TEMPLATE.md`；`docs/fork/branch-and-sync-sop.md` | 所有现有 workflow 使用 GitHub 托管 `ubuntu-latest` **Linux amd64**；`ci.yml` 在上游的 PR 触发之外增加推送 `main` 触发（本仓日常改动不走 PR）；本仓 PR 不配置 anti-slop 自动标签、评论或关闭门禁，允许 AI-generated 与 AI-assisted 贡献；上游贡献独立遵循目标仓当时规则；**tag → GHCR** 仅 `:<version>`（tip 时再加 `:latest`），无冗余 `*-amd64` 正式 tag；构建与签名后复核 main tip，只有仍匹配的 tag 才同时提升 Docker `:latest` 与 GitHub `/releases/latest`；镜像与手动二进制 workflow 以共享 tag 锁串行写 Release，重建已有 Release 保留其 latest 状态，新建历史 Release 明确不提升；`cache-to` **ignore-error**；**禁止** Docker Hub `calciumion/new-api`；裸二进制仅 `release.yml` **workflow_dispatch**；electron 构建对本仓禁用 | 公开仓统一使用 GitHub 官方一次性 runner，避免维护自托管 runner、runner group 可见性和离线排队问题；本仓贡献方式不由 anti-slop 自动判定，但不改写上游政策；部署走镜像；更新检查依赖 `releases/latest`；避免污染上游 Hub；cache 故障不拖垮已 push 镜像 | medium（上游常改 workflow；同步时须保留 GitHub-hosted Linux amd64 与本仓 GHCR 策略） | 推送 `main` 后 `ci.yml` 对该提交运行后端与前端检查；`AkumaRealLabs/ggapi` 新 PR 不运行 `pr-quality` / anti-slop，不因 AI 生成或辅助而自动标记、评论或关闭；本仓 PR 保留模板结构、验证记录可选，执行 git 身份与条件 AI 披露，并在 Fork Diff Inventory 段二选一声明永久差异；向 `QuantumNous/new-api` 贡献时 Mode G 同时读取上游 Git tree 定义与 GitHub API 运行态，核对 workflow `state`、classic branch protection、rulesets、effective branch rules 与 required contexts；tag、branch image、手动 bare release 与 Electron 禁用提示任务均不引用 self-hosted runner；推 fork tag → 仅 `IMAGE:TAG`，且仅构建后 main tip 仍匹配时增加 Docker `:latest` 并设为 GitHub latest Release；main 前进时两者都不提升；同 tag 的 Docker 与 bare binary workflow 不并行写 Release；手动重建当前 latest tag 后 `/releases/latest` 不变，补建历史 Release 不会提升 latest；不默认 bare 二进制；branch 镜像 `branch-*-hash`；上游同步后仍为 `ghcr.io/...` | active | Akuma-real | 2026-07-11 | `chore/org-linux-runner` / `docs/version-tag-scheme` / `chore/tag-ghcr-no-binary` / `fix/ghcr-cache-ignore-error-single-tags` / `sync/upstream-20260713` / `chore/pr-check-hosted-runner` #32 / `chore/manual-review-hosted-runners` / `docs/remove-anti-slop-policy` / `docs/ggapi-fork-no-pr` |
| GG-004 | feature | `setting/system_setting/update_check.go`；`controller/update_check*.go`；`service/public_http_client*.go`；`model/option.go`；`router/api-router.go`；`middleware/access_token_routes.go`；`web/src/features/system-update/{api,releases}.ts` 及 `__tests__/`；`web/src/features/system-settings/{maintenance,operations}/**`；`web/src/features/system-settings/types.ts`；`web/src/i18n/locales/**` | 检查更新改为服务端代理；后台可配置 Release API URL + GitHub PAT（私有仓）；PAT 仅发送到受信任的 `https://api.github.com`，重定向保持同一信任边界，所有目标在 URL 校验及拨号时拒绝私网/回环/链路本地地址；默认指向 `AkumaRealLabs/ggapi`。2026-10-13 起沿用上游 `system-update`（顶栏提示、每小时自动检查、发布详情弹窗），只把数据源换成 `GET /api/option/check_update`：接口对所有管理员开放（改源配置仍仅 Root），统一返回发布列表（兼容 `…/releases/latest` 单对象与 `…/releases` 列表），GitHub 限流映射为 HTTP 429，失败时只返回上游界面已有的三类消息键（检查失败、限流、发布数据异常），URL 被拒、PAT/404、HTTP 状态与返回内容异常等具体原因写入服务端日志（`common.LogText`，有中文），访问令牌规则为 `option:read`；版本比较接受本仓 `v<上游 tag>.N`（如 `v1.0.0-rc.25.1` 排在 `v1.0.0-rc.25` 之后、`v1.0.0-rc.26` 之前）；发布链接优先用服务端返回的 `html_url`，缺省时沿用上游链接；维护页在上游版本卡片与检查入口下保留 URL/PAT 配置表单，保存后刷新检查结果 | 私有仓浏览器直连 GitHub 无法鉴权；PAT 不得下发前端或发送到非 GitHub 主机；可配置 URL 不得形成 SSRF；上游默认只检查官方仓，本仓发版与 tag 格式不同 | medium（上游改 system-update、update-checker、HTTP 客户端、Option API 或访问令牌路由表时冲突） | 管理员登录后顶栏按本仓 Release 提示新版本，普通用户不发请求；Root 保存 URL/PAT 后检查结果随之刷新；GetOptions 不返回 Token；GitHub 限流时前端提示限流而非网络错误；检查失败时服务端日志写明原因且不含 PAT 与 URL 凭据；`v1.0.0-rc.25.1` < `v1.0.0-rc.25.2` < `v1.0.0-rc.26`；配置 PAT 时 HTTP、非 `api.github.com` 主机及跨主机重定向均被拒；字面量及 DNS 解析到私网/回环/链路本地的目标均不会连接；`go test ./controller ./router`、`vitest src/features/system-update` | active | Akuma-real | 2026-07-11 | `feat/update-check-url-pat` / `sync/upstream-20260713` / `sync/upstream-20260727` / `sync/upstream-20261013` |
| GG-007 | feature / patch | `common/{constants.go,log_text.zh-CN.json}`；`model/{affiliate*,db_error*,main,option,redemption*,subscription,task_cas_test,topup,user}.go`；`controller/{affiliate,option,user}.go`；`router/api-router.go`；`middleware/access_token_routes.go`；`web/src/features/{affiliate,wallet,users,system-settings}/**`；`web/src/i18n/locales/**` | 新增一级邀请返佣：在线充值、外部支付订阅与兑换码成功兑换后按默认或邀请人自定义比例结算到 `aff_quota`，以来源唯一账单、订单/兑换事务锁和同事务更新保证幂等。2026-10-13 起直接补丁接入上游支付完成函数，不再保留本仓 `completeTopUp` 包装：五个充值渠道与管理员补单在上游 `creditTopUpQuota` 之前调用 `settleTopUpAffiliateCommissionTx`（与入账同事务）；外部订阅在 `CompleteSubscriptionOrder` 锁用户行之前结算，保持 `SettleAffiliateCommissionTx` 按 id 升序锁 invitee/inviter 的锁序；兑换码在上游 `Redeem` 事务内结算，邀请关系变化时整单重试（最多 3 次）；返佣接口已登记访问令牌规则（用户侧 `wallet:read/write`，管理侧 `user:read/write` 与 `billing:read`）；提供用户一次性补绑、管理员改绑/解绑、邀请码与比例管理、返佣明细及全局账单 | 现有固定注册邀请奖励无法覆盖持续充值、订阅与兑换推广，需要可审计、不可重复结算且不影响余额订阅或管理员赠送的独立资金路径 | high（充值/订阅完成事务、兑换事务、用户额度与关系锁、数据库迁移及管理端资金视图） | SQLite/MySQL/PostgreSQL 迁移与唯一索引；五种支付渠道和管理员补单只返佣一次；外部订阅返佣且余额订阅不返；兑换码按兑换时邀请关系以 `RED-<id>` 安全引用返佣且不暴露密钥，重复兑换不重复返佣，返佣写入失败时兑换码/用户额度/邀请人返利全回滚，并发改绑在用户行锁后复核关系，变化则整单回滚并以新事务重试，旧邀请人不会收到返佣；默认/自定义/0%/合规关闭/禁用邀请人及返佣余额溢出按规则跳过且兑换成功；自邀与循环拦截、改绑仅影响后续订单/兑换；账单/订单/用户额度失败全回滚；`aff_quota` 转账后缓存立即可用；用户与管理端来源筛选/展示及七语言键；返佣与邀请码相关报错走 `common.NewMessage` 消息键，钱包绑定与后台邀请设置弹窗按界面语言显示；`router` 访问令牌路由覆盖测试通过 | active | Akuma-real | 2026-07-14 | `feat/affiliate-commission` / `feat/redemption-affiliate-commission` / `sync/upstream-20260813` / `sync/upstream-20261013` |
| GG-010 | patch | `relay/official_responses_penalty*.go`；`relay/{chat_completions_via_responses,responses_request}.go` | Chat Completions 与直接 Responses 请求经官方 OpenAI/Azure 路由时移除 `frequency_penalty` / `presence_penalty`；兼容供应商路由仍保留显式值。2026-10-13 起逻辑集中在 `relay/official_responses_penalty.go`，上游 `relayResponsesRequest` 与 `PrepareResponsesRequest`（含请求体透传分支）各调用一次 | 官方 Responses API 不支持 penalty 字段，原样转发会使合法请求被上游拒绝 | low（仅官方 Responses 渠道兼容清理；上游改 Responses 请求准备流程时需重接调用点） | OpenAI/Azure 转换结果与透传请求体均不含两个 penalty 字段；Advanced Custom 等兼容路由保留显式 `0` 与非零值；`go test ./relay/...` | active | Akuma-real | 2026-08-15 | `sync/upstream-20260813` / `sync/upstream-20261013` |


### 已关闭（可选归档）

| ID | 最终状态 | 关闭日期 | 说明 |
|----|----------|----------|------|
| GG-005 | dropped | 2026-07-27 | 第三前端壳、classic、多主题与独立视觉重构在同步 `v1.0.0-rc.22` 时移除；仓库回归官方单前端 `web/`，GG-004/006/007 以最小功能补丁直接维护。 |
| GG-006 | dropped | 2026-10-13 | 同步 `v1.0.0-rc.44` 时按维护者决定删除“仅会员权益套餐”前后端（字段、接口、前端、测试与翻译键）。旧库保留 `membership_only` 列（GORM 不删列），启动迁移 `retireMembershipSubscriptions`（`model/membership_retirement.go`）一次性清理：会员套餐禁用、其 pending 订单过期、`scheduled` 订阅取消、生效中的会员订阅立即到期并由上游到期任务回退分组；处理过的行同时清除 `membership_only`，重启不会重复执行，也不会覆盖之后的管理员操作。注意：会员套餐 `total_amount=0` 在上游语义下是“不限额度”，不要重新启用这些套餐；已过期的会员 pending 订单若之后才收到支付回调会完成失败，需人工处理。 |
| GG-008 | dropped | 2026-10-13 | 资金安全补丁（跨分组补扣、Midjourney 先扣后请求、额度立即落库、Token owner lease、余额不足拦截等）改用上游 `v1.0.0-rc.44` 的实现，本仓补丁全部回退。 |
| GG-009 | dropped | 2026-10-13 | 渠道 HTTP 协议/分片设置在旁路请求中的透传补丁全部回退，改用上游实现。上游 `rc.44` 的任务轮询接口 `FetchTask` 仍只接收代理字符串，渠道 `HTTPProtocol` / `HTTP2ConnectionShards` 在任务轮询等旁路不生效。 |
| GG-011 | dropped | 2026-10-13 | 入账上限与退款事务改用上游实现：钱包上限为上游 `common.MaxWalletQuota`（2^53−1），不再执行本仓 int32 条件更新；退款与终态处理以上游代码为准。 |
| GG-012 | dropped | 2026-10-13 | Advanced Custom 编辑器的翻译与校验文案改用上游版本，本仓补丁回退。 |

---

## 6. 同步后检查清单（配合 SOP）

每次 `sync/upstream-*` 合并前自检：

1. [ ] `git fetch upstream` 后记录将合入的 upstream 范围（tag/commit）
2. [ ] 所有 `active` / `needs-rebase` 项在冲突解决后行为仍正确
3. [ ] 冲突中默认采纳的上游安全/计费/协议修复已保留
4. [ ] 本表元信息（基准版本/commit、日期）已更新
5. [ ] 新增的本仓-only 改动已补登记
6. [ ] 已 `upstreamed` / `dropped` 的补丁未再次被误加回

---

## 7. 相关文档

- [分支命名与同步 SOP](./branch-and-sync-sop.md)
- [工程约定 AGENTS.md](../../AGENTS.md)
- [计费表达式说明](../../pkg/billingexpr/expr.md)（改计费前必读）
