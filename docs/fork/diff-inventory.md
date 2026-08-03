# ggapi 二开差异清单

用于登记本仓库相对 **upstream（QuantumNous/new-api）** 的长期定制，降低同步冲突成本，并保证发版可回归。

关联流程见 [分支命名与同步 SOP](./branch-and-sync-sop.md)。

---

## 1. 元信息

| 字段 | 填写说明 | 当前值 |
|------|----------|--------|
| 清单维护人 | 负责督促更新的人 | Akuma-real |
| 最近更新日期 | `YYYY-MM-DD` | 2026-08-02 |
| 基准 upstream 版本 | 最近一次完整同步时的 tag 或 `VERSION` | **`v1.0.0-rc.23`**（`0ab020206`；官方前端收敛为 `web/`，本仓仅重放 GG-004/006/007 的必要功能补丁。发版 `N` 自 `.1` 起重计） |
| 基准 upstream commit | 最近一次 `merge upstream/main` 的提交 SHA（短哈希即可） | `0ab020206` |
| 本仓版本策略 | git tag 格式（见 [SOP §5.1](./branch-and-sync-sop.md)） | **`v<上游基线>.N`**（例已发 `v1.0.0-rc.20.5`；同步后新基线发版从 **`v1.0.0-rc.22.1`** 起）；基线须为 `origin/main` 祖先；正式 tag **只钉** `origin/main` tip；**禁止**与 upstream 同名 tag；`N` 仅在「基准 upstream 版本」串变化时归 1；**默认 tag 产物 = GHCR 镜像**（非裸二进制） |

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
| GG-002 | docs | `.agents/skills/ggapi-fork/`；`docs/fork/README.md` / `AGENTS.md` 入口链接 | `/ggapi-fork` 教练 skill **v2.0.0**（Modes A–I + 可控自升级 + C2-pre 用户手动终端 Agent 审查 + **C-continue** + 链式 C→F + 官方单前端 + **tag `v<上游>.N` → GHCR wait**）；含 oxlint 发车、merge flake 校验、GitHub-hosted runner、检查更新 PAT、i18n zh-TW 等 | 二开需要命令级教练，且 skill 须能跟进文档/实战而不静默削弱安全规则 | low（独立 skill 目录；入口链接触及 `AGENTS.md`/`docs/fork` 为 medium 旁路） | `/ggapi-fork` 可发现；前端统一为官方 `web/`，GG-005 已退役；tag→GHCR 见 SOP §5.1 / GG-003；提交前当前 Agent 只准备并指纹化最终树、提示用户在新终端用自选 Agent 审查并等待结果，不调用任何审查工具；落后分支按 merge base 审查，交接时工作区无未暂存/未跟踪内容，接收结果前重新 fetch 并校验记录的 main tip；C2 前校验 index 快照，C2 后改按 `HEAD^{tree}` + clean workspace 校验，正常 staged → clean 不会误判 stale；findings 修复后再次等待用户手动复审，stale result / merge pin 防护仍有效；跨会话回传结果只恢复暂停前已授权的链，不补 push/PR/merge/发版词；PR 已开时「继续」不重写代码且不补 push 词；自升级 L1 可小补、L2/L3 须确认、不自动 commit；文档优先于 skill | active | Akuma-real | 2026-07-11 | `docs/fork-sop` / `docs/ggapi-fork-*` / `docs/version-tag-scheme` / `chore/tag-ghcr-no-binary` / `docs/ggapi-fork-1.6.0` / `docs/ggapi-fork-1.7.0` / `chore/manual-review-hosted-runners` / `sync/upstream-20260727` / `docs/remove-anti-slop-policy` |
| GG-003 | infra | `.github/workflows/*`（`docker-build.yml`、`docker-image-branch.yml`、`release.yml`、`electron-build.yml`）；`.github/PULL_REQUEST_TEMPLATE.md`；`docs/fork/branch-and-sync-sop.md` | 所有现有 workflow 使用 GitHub 托管 `ubuntu-latest` **Linux amd64**；本仓 PR 不配置 anti-slop 自动标签、评论或关闭门禁，允许 AI-generated 与 AI-assisted 贡献；上游贡献独立遵循目标仓当时规则；**tag → GHCR** 仅 `:<version>`（tip 时再加 `:latest`），无冗余 `*-amd64` 正式 tag；构建与签名后复核 main tip，只有仍匹配的 tag 才同时提升 Docker `:latest` 与 GitHub `/releases/latest`；镜像与手动二进制 workflow 以共享 tag 锁串行写 Release，重建已有 Release 保留其 latest 状态，新建历史 Release 明确不提升；`cache-to` **ignore-error**；**禁止** Docker Hub `calciumion/new-api`；裸二进制仅 `release.yml` **workflow_dispatch**；electron 构建对本仓禁用 | 公开仓统一使用 GitHub 官方一次性 runner，避免维护自托管 runner、runner group 可见性和离线排队问题；本仓贡献方式不由 anti-slop 自动判定，但不改写上游政策；部署走镜像；更新检查依赖 `releases/latest`；避免污染上游 Hub；cache 故障不拖垮已 push 镜像 | medium（上游常改 workflow；同步时须保留 GitHub-hosted Linux amd64 与本仓 GHCR 策略） | `AkumaRealLabs/ggapi` 新 PR 不运行 `pr-quality` / anti-slop，不因 AI 生成或辅助而自动标记、评论或关闭；本仓 PR 保留模板结构、验证记录可选，执行 git 身份与条件 AI 披露，并在 Fork Diff Inventory 段二选一声明永久差异；向 `QuantumNous/new-api` 贡献时 Mode G 同时读取上游 Git tree 定义与 GitHub API 运行态，核对 workflow `state`、classic branch protection、rulesets、effective branch rules 与 required contexts；tag、branch image、手动 bare release 与 Electron 禁用提示任务均不引用 self-hosted runner；推 fork tag → 仅 `IMAGE:TAG`，且仅构建后 main tip 仍匹配时增加 Docker `:latest` 并设为 GitHub latest Release；main 前进时两者都不提升；同 tag 的 Docker 与 bare binary workflow 不并行写 Release；手动重建当前 latest tag 后 `/releases/latest` 不变，补建历史 Release 不会提升 latest；不默认 bare 二进制；branch 镜像 `branch-*-hash`；上游同步后仍为 `ghcr.io/...` | active | Akuma-real | 2026-07-11 | `chore/org-linux-runner` / `docs/version-tag-scheme` / `chore/tag-ghcr-no-binary` / `fix/ghcr-cache-ignore-error-single-tags` / `sync/upstream-20260713` / `chore/pr-check-hosted-runner` #32 / `chore/manual-review-hosted-runners` / `docs/remove-anti-slop-policy` |
| GG-004 | feature | `setting/system_setting/update_check.go`；`controller/update_check*.go`；`service/public_http_client*.go`；`model/option.go`；`router/api-router.go`；`web/src/features/system-settings/{maintenance,operations}/**`；`web/src/features/system-settings/types.ts`；`web/src/i18n/locales/**` | 检查更新改为服务端代理；后台可配置 Release API URL + GitHub PAT（私有仓）；PAT 仅发送到受信任的 `https://api.github.com`，重定向保持同一信任边界，所有目标在 URL 校验及拨号时拒绝私网/回环/链路本地地址；默认指向 `AkumaRealLabs/ggapi`；前端使用官方 Base UI/Form 组件 | 私有仓浏览器直连 GitHub 无法鉴权；PAT 不得下发前端或发送到非 GitHub 主机；可配置 URL 不得形成 SSRF | medium（上游改 update-checker、HTTP 客户端或 Option API 时冲突） | Root 后台保存 URL/PAT 后「检查更新」走 `/api/option/check_update`；GetOptions 不返回 Token；公开仓可不填 PAT并使用公网 HTTP(S) URL；配置 PAT 时 HTTP、非 `api.github.com` 主机及跨主机重定向均被拒；无论全局 fetch 设置如何，字面量及 DNS 解析到私网/回环/链路本地的目标均不会连接 | active | Akuma-real | 2026-07-11 | `feat/update-check-url-pat` / `sync/upstream-20260713` / `sync/upstream-20260727` |
| GG-006 | feature / patch | `common/quota_math*.go`；`model/{subscription*,token,db_time,main,task_cas_test}.go`；`service/{billing_session,funding_source,group*,subscription_reset_task,task_billing_test}.go`；`controller/subscription*.go`；`controller/token.go`；`web/src/features/subscriptions/**`；`web/src/features/wallet/**`；`web/src/lib/currency.ts`；`web/src/i18n/locales/**`；`web/scripts/{sync-i18n.mjs,subscription-purchase-dialog.test.tsx}` | 新增“仅会员权益套餐”：购买后限时切换用户分组；不改写用户 Token 分组；不发放订阅额度、不参与预扣或额度重置，API 始终从钱包扣费；到期按套餐配置回退分组；购买前按 `plan → user` 锁顺序串行化会员 checkout，锁内复核套餐启用状态，允许有效会员在到期前自助续费并顺延为 `scheduled`，统一拦截任意会员套餐 pending 订单（预占队列名额）、3 份 scheduled 队列上限、启用中的空分组 Token 和终身购买上限；自助订阅接口下发队列上限与会员 pending 数，前端按服务端上限展示并预拦截；余额购买按原有向上取整语义使用 `QuotaFromDecimalChecked`，超范围直接拒绝且不扣款，成功响应返回 `active/scheduled`；外部支付入口与拉起失败统一返回稳定英文业务键，由购买弹窗单点翻译和 toast，不再依赖旧式 `{message:"error", data:"..."}`；管理端绑定、失效与删除结果均返回稳定英文键，由前端翻译；已付款订单履约不重复执行 checkout 拦截，仅保留购买上限；`GroupSpecialUsableGroup` 删除规则可隐藏用户身份分组，避免把会员身份当作 Token 路由分组；购买金额使用站点本地货币符号并固定两位小数；更新套餐时省略 `membership_only` 不误降级（事务内以锁行解析）；有 active/scheduled 订阅或未完成 pending 订单时禁止切换类型；创建订阅与改类型共享 plan 行锁；支付拉起失败统一过期 pending；`reset=never` 时清理 due 的 `next_reset_time` 防 worker 死循环；i18n 同步保留 key 文本顺序及明文项目归因 key | VIP 需要作为可自助购买的限时折扣权益存在，并与余额充值、订阅额度套餐、Token 渠道路由严格分离 | high（订阅模型/迁移、支付事务、Token 权限、计费选择与用户分组缓存均为资金相关热点） | 三库迁移新增字段且旧记录回填 false；外部 checkout 在 plan 锁内遇到禁用套餐必须拒绝；有效会员可在到期前续费入队；跨套餐会员 pending（预占队列名额）、3 份 scheduled 上限、启用中的空分组 Token 和终身上限均在创建支付订单或余额扣款前被拒，失败时钱包/订单/订阅不变，禁用或软删除的空分组 Token 不阻止购买；自助接口返回服务端队列上限和会员 pending 数，前端队满或存在 pending 时禁用支付；余额金额换算保持 ceil，超 int32 范围拒绝而非饱和扣费，成功 toast 按 `active/scheduled` 区分立即生效与排队；特殊删除规则隐藏测试身份分组且保留测试路由分组，不改已有 Token；`subscription_first` / 历史 `subscription_only` 均扣钱包且 `AmountUsed` 不变；额度重置被拒绝；scheduled 不提前切组，worker 按 expire → activate scheduled → reset quota 运行，提前失效 active 会立即启用队首，取消/删除 scheduled 会重排，管理端与钱包显示 Queued；最终到期回退；省略 `membership_only` 的更新保留原类型；有 active/scheduled 订阅时改类型被拒；plan reset=never 的 due 订阅被清 schedule 且二次 worker 不再选中；四种外部支付渠道的参数、套餐状态、渠道配置、用户、回调、支付方式及拉起失败均返回可翻译 `message`，官方前端购买端只 toast 一次，管理端操作消息也经 `t()`，并覆盖七种语言；自定义 `¥`、汇率 1 时显示 `¥19.90`，普通 CUSTOM 金额保持非固定两位且无符号空格；`i18n:sync` 不移动数字 key，`footer.newapi.projectAttributionSuffix` 保持明文 | active | Akuma-real | 2026-07-15 | `fix/subscription-group-billing` / `feat/vip-membership-plan` / `fix/vip-membership-hardening` |
| GG-007 | feature / patch | `common/{constants,quota_math*}.go`；`model/{affiliate*,db_error*,main,option,redemption*,subscription,task_cas_test,topup,user}.go`；`controller/{affiliate,option,topup,user}.go`；`router/api-router.go`；`web/src/features/{affiliate,wallet,users,system-settings}/**`；`web/src/i18n/locales/**` | 新增一级邀请返佣：在线充值、外部支付订阅与兑换码成功兑换后按默认或邀请人自定义比例结算到 `aff_quota`，以来源唯一账单、订单/兑换事务锁和同事务更新保证幂等；提供用户一次性补绑、管理员改绑/解绑、邀请码与比例管理、返佣明细及全局账单 | 现有固定注册邀请奖励无法覆盖持续充值、订阅与兑换推广，需要可审计、不可重复结算且不影响余额订阅或管理员赠送的独立资金路径 | high（充值/订阅完成事务、兑换事务、用户额度与关系锁、数据库迁移及管理端资金视图） | SQLite/MySQL/PostgreSQL 迁移与唯一索引；五种支付渠道和管理员补单只返佣一次；外部订阅返佣且余额订阅不返；兑换码按兑换时邀请关系以 `RED-<id>` 安全引用返佣且不暴露密钥，重复兑换不重复返佣，返佣写入失败时兑换码/用户额度/邀请人返利全回滚，并发改绑在用户行锁后复核关系，变化则整单回滚并以新事务重试，旧邀请人不会收到返佣；默认/自定义/0%/合规关闭/禁用邀请人及返佣余额溢出按规则跳过且兑换成功；自邀与循环拦截、改绑仅影响后续订单/兑换；账单/订单/用户额度失败全回滚；`aff_quota` 转账后缓存立即可用；用户与管理端来源筛选/展示及七语言键 | active | Akuma-real | 2026-07-14 | `feat/affiliate-commission` / `feat/redemption-affiliate-commission` |
| GG-008 | patch | `model/{errors,user,user_update_test,utils}.go`；`relay/{common/relay_info,mjproxy_handler*}.go`；`service/{billing,billing_session*,funding_source,quota,task_billing*,tiered_settle*}.go`；`web/src/features/keys/components/{api-key-group-cell.tsx,__tests__/api-key-group-cell.test.tsx}` | Auto 跨分组重试切换到更高倍率分组时，在发送上游请求前以共享数据库条件更新原子复核并补扣钱包额度；钱包请求不再以信任额度跳过预扣，Midjourney 固定价任务也在发送前完成钱包 CAS 与 Token 预留，只有成功落库的已扣款任务才保留可退款 `Quota`；同步结算、兼容结算与异步任务差额补扣同样复用余额门禁；`User.Quota` 不再进入进程内批量 map，所有钱包增减直接持久化，Redis 仅作为可失效缓存；余额不足返回 `insufficient_user_quota` 且不形成欠费；API key 列表始终显示独立 `Auto` 标识，仅在 `cross_group_retry=true` 时追加 `Cross-group` 状态 | 上游 `rc.23` 的补充预扣与结算差额允许钱包变负，信任额度与 Midjourney 的先请求后扣款路径在并发 CAS 失败时还会漏扣或产生无扣款退款；前端同时无条件显示跨分组状态；进程内 pending map 无法作为多节点共享资金账本，短 TTL 缓存缺失还会导致长请求退款或异步结算永久漏记 | high（触及请求发送前的计费预扣、同步/异步结算、Midjourney 任务退款与钱包额度持久化；前端显示本身为 low） | `BATCH_UPDATE_ENABLED=true` 时钱包扣款/退款仍立即落库，缓存缺失不会漏记且 batch flusher 不会重复应用；并发请求及高倍率重试只有余额足够的预扣成功，其余在发送上游前返回 `ErrorCodeInsufficientUserQuota`；高余额钱包仍按估算值真实预扣；Midjourney 上游失败或任务落库失败会退还预扣，未扣款任务的 `Quota=0`，任务失败不会凭空退款；同步结算与异步任务差额补扣余额不足时保留原余额和结算状态，数据库余额不为负；SQLite/MySQL/PostgreSQL 均使用 `UPDATE ... WHERE quota >= ?` 作为跨节点 CAS；批量刷新不增加钱包全局锁；Auto key 在倍率缺失且跨分组重试关闭时仍显示 `Auto`，`cross_group_retry=true/false` 分别显示/隐藏 `Cross-group` | active | Akuma-real | 2026-08-02 | `sync/upstream-20260802` #39 |
| GG-009 | patch | `controller/{channel,channel-billing,channel_upstream_update,codex_usage,midjourney,video_proxy*}.go`；`relay/{relay_task,mjproxy_handler*}.go`；`relay/channel/{adapter,ali,baidu,dify,gemini,ollama,replicate,task/**,vertex/service_account}.go`；`service/{codex_channel_models,midjourney,task_polling*}.go` | 将渠道级 `HTTPProtocol` / `HTTP2ConnectionShards` 与代理设置传递到模型发现、余额/用量查询、渠道上传、异步任务轮询及辅助上游请求；任务轮询接口统一接收完整 `ChannelSettings`，不再只传代理字符串；Midjourney image-seed 与原任务 action 会用原渠道完整覆盖 key、代理和协议上下文 | 上游 `rc.23` 仅在部分主 relay 请求选择策略感知 client，管理端配置在模型同步和异步任务等旁路中被静默忽略；Midjourney 绕过选渠或切回原渠道时还可能缺失设置或沿用错误渠道设置 | medium（触及多个渠道适配器的 HTTP client 选择与任务轮询接口） | `http1` 与多分片设置在通用/Gemini/Codex/Ollama 模型发现、Suno/视频任务轮询及 Vertex token 获取中生效；Midjourney image-seed 与 action 使用原任务渠道的 key、代理、HTTP 协议和分片设置；轮询测试断言完整 `ChannelSettings` 从数据库渠道传到适配器；默认设置保持原有 client 池语义；`go test ./controller ./relay/... ./service` | active | Akuma-real | 2026-08-02 | `sync/upstream-20260802` #39 |

**GG-006 会员顺延补充（2026-07-15）：** 同一用户最多一份未到期 `active` 会员；有效期内自助续费、已付款回调或管理员赠送遇到现有会员时创建 `scheduled`，pending 预占名额、最多 3 份并按队尾顺延且不提前切组/写 `PrevUserGroup`。维护任务对额度订阅走通用 expire；会员在 `ActivateDueMembershipSubscriptions` 内同事务 expire→activate，避免中间态用基础分组路由；删除用户导致的孤儿订阅在锁用户失败时清理并跳过，不拖垮整轮 worker。abandoned pending 超过 24h TTL 在 checkout/自助计数前自动过期；有效会员期间创建/更新/启用空分组 Token 被拒，与购买前空分组拦截一致；Token 新增/更新与会员 checkout 共用 user 行锁并在同一事务内校验后写入，并发交错最多一侧成功，不会同时留下会员状态与启用的空分组 Token。`scheduled` 与 active 一并阻止套餐类型切换；SQLite 状态机测试覆盖支付回调竞态、不同会员分组顺序激活、最终回退、队列重排和 worker 幂等；生产串行化依赖 MySQL/PostgreSQL 的 `FOR UPDATE`。套餐 CRUD、赠送、失效/删除和额度重置的业务失败统一返回稳定英文键，官方管理端通过 `skipBusinessError + t(message)` 单点 toast；钱包列表、管理表格、赠送选择器和购买弹窗统一使用 `formatLocalCurrencyAmount`。

### 已关闭（可选归档）

| ID | 最终状态 | 关闭日期 | 说明 |
|----|----------|----------|------|
| GG-005 | dropped | 2026-07-27 | 第三前端壳、classic、多主题与独立视觉重构在同步 `v1.0.0-rc.22` 时移除；仓库回归官方单前端 `web/`，GG-004/006/007 以最小功能补丁直接维护。 |

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
