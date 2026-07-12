# ggapi 二开差异清单

用于登记本仓库相对 **upstream（QuantumNous/new-api）** 的长期定制，降低同步冲突成本，并保证发版可回归。

关联流程见 [分支命名与同步 SOP](./branch-and-sync-sop.md)。

---

## 1. 元信息

| 字段 | 填写说明 | 当前值 |
|------|----------|--------|
| 清单维护人 | 负责督促更新的人 | Akuma-real |
| 最近更新日期 | `YYYY-MM-DD` | 2026-07-12 |
| 基准 upstream 版本 | 最近一次完整同步时的 tag 或 `VERSION` | **`v1.0.0-rc.20`**（`ad900bbb` ≈ `v1.0.0-rc.20-32-gad900bbb`；`v1.0.0-rc.21` **尚非** 本仓 `main` 祖先，勿当当前基线。上游 tip 可能已更新，完整同步后再改此行） |
| 基准 upstream commit | 最近一次 `merge upstream/main` 的提交 SHA（短哈希即可） | `ad900bbb` |
| 本仓版本策略 | git tag 格式（见 [SOP §5.1](./branch-and-sync-sop.md)） | **`v<上游基线>.N`**（例已发 `v1.0.0-rc.20.1`）；基线须为 `origin/main` 祖先；正式 tag **只钉** `origin/main` tip；**禁止**与 upstream 同名 tag；`N` 仅在「基准 upstream 版本」串变化时归 1；**默认 tag 产物 = GHCR 镜像**（非裸二进制） |

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
| GG-002 | docs | `.agents/skills/ggapi-fork/`；`docs/fork/README.md` / `AGENTS.md` 入口链接 | `/ggapi-fork` 教练 skill **v1.5.1**（Modes A–I + 可控自升级 + C2-pre + 第三壳 + **tag `v<上游>.N` → GHCR**）；含 org-linux、检查更新 PAT、i18n zh-TW 等 | 二开需要命令级教练，且 skill 须能跟进文档/实战而不静默削弱安全规则 | low（独立 skill 目录；入口链接触及 `AGENTS.md`/`docs/fork` 为 medium 旁路） | `/ggapi-fork` 可发现；默认前端 `web/ggapi`；tag→GHCR 见 SOP §5.1 / GG-003；C2-pre 可走通；自升级 L1 可小补、L2/L3 须确认、不自动 commit；文档优先于 skill | active | Akuma-real | 2026-07-11 | `docs/fork-sop` / `docs/ggapi-fork-*` / `docs/version-tag-scheme` / `chore/tag-ghcr-no-binary` |
| GG-003 | infra | `.github/workflows/*`（`docker-build.yml`、`docker-image-branch.yml`、`release.yml`） | CI `org-linux` + **Linux amd64**；**tag → GHCR**（`ghcr.io/<owner>/<repo>`）+ **Release 元数据**（供更新检查）；**禁止** Docker Hub `calciumion/new-api`；裸二进制仅 `release.yml` **workflow_dispatch** | 组织 runner；部署走镜像；更新检查依赖 `releases/latest`；避免污染上游 Hub | medium（上游常改 workflow） | 推 fork tag → GHCR + GitHub Release 页；不默认上传 bare 二进制；branch 镜像 `branch-*-hash`；上游同步后 image 仍为 `ghcr.io/...` | active | Akuma-real | 2026-07-11 | `chore/org-linux-runner` / `docs/version-tag-scheme` / `chore/tag-ghcr-no-binary` |
| GG-004 | feature | `setting/system_setting/update_check.go`；`controller/update_check.go`；`model/option.go`；`router/api-router.go`；`web/default/.../update-checker-section.tsx`（及 `web/ggapi` 对应页，功能追 default 后） | 检查更新改为服务端代理；后台可配置 Release API URL + GitHub PAT（私有仓）；默认指向 `AkumaRealLabs/ggapi` | 私有仓浏览器直连 GitHub 无法鉴权；PAT 不得下发前端 | medium（上游改 OtherSetting/update-checker 时冲突） | Root 后台保存 URL/PAT 后「检查更新」走 `/api/option/check_update`；GetOptions 不返回 Token；公开仓可不填 PAT | active | Akuma-real | 2026-07-11 | `feat/update-check-url-pat` |
| GG-005 | feature / infra | `web/ggapi/**`；`web/package.json` workspaces；`makefile`（`build-web-ggapi` / `dev-web-ggapi`，`dev`→ggapi）；`Dockerfile` `builder-ggapi`；`.github/workflows/docker-build.yml`（tag→GHCR 三壳）；可选 `release.yml`（手动裸二进制）；`main.go` embed；`common/constants.go` / `embed-file-system.go`；`router/web-router.go`；`setting/system_setting/theme.go`（默认 `ggapi`）；`controller/option.go` 校验；`AGENTS.md` / `docs/fork/*` 第三壳与追 default 说明 | 第三前端壳 `web/ggapi`（自 default 复制起步）；运行时 `theme.frontend=ggapi`；构建/embed/Docker 全链路；**功能长期追 `web/default`**，视觉与产品定制落在 ggapi | 避免在 `web/default` 上做永久大改导致上游同步痛苦；独立壳便于皮肤与产品化 | high（三壳源码树 + 多处接线；上游改 theme/embed/Docker/makefile 时需重放） | `theme.frontend` 可选 `ggapi\|default\|classic`；默认 ggapi；Dockerfile 三壳齐全；tag 推 GHCR；同步后 default 新功能有 port 到 ggapi 的 follow-up | active | Akuma-real | 2026-07-11 | `feat/web-ggapi-shell` #7 |

### 已关闭（可选归档）

| ID | 最终状态 | 关闭日期 | 说明 |
|----|----------|----------|------|
| （无） | — | — | — |

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
