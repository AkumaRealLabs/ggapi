#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const SHIP_FIELDS = [
  '当前阶段',
  '当前分支',
  'Commit',
  'Push',
  'PR',
  'Merge',
  'diff-inventory',
  '验证',
  '下一步授权',
];

const STAGE_ORDER = ['B', 'C1', 'C2-pre', 'C2', 'C3', 'C4', 'C5', 'F'];

function git(cwd, args, spawn = spawnSync) {
  const result = spawn('git', args, {
    cwd,
    encoding: 'utf8',
    timeout: 2000,
    env: { ...process.env, GIT_OPTIONAL_LOCKS: '0', LC_ALL: 'C' },
  });
  if (result.error || result.status !== 0) return null;
  return String(result.stdout ?? '').trim();
}

function parseCount(value) {
  const parsed = Number.parseInt(value ?? '', 10);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function sanitizeRemoteUrl(value) {
  if (typeof value !== 'string' || value.length === 0) return null;

  // URL treats scp-style remotes such as user:secret@host:path as a custom
  // scheme, so strip the credential prefix before attempting URL parsing.
  const withoutQuery = value.replace(/[?#].*$/, '');
  const scpSafe = withoutQuery.replace(/^[^/\s@]+@(?=[^/\s:]+:)/, '');
  if (scpSafe !== withoutQuery) return scpSafe;

  try {
    const parsed = new URL(value);
    parsed.username = '';
    parsed.password = '';
    parsed.search = '';
    parsed.hash = '';
    return parsed.toString();
  } catch {
    return withoutQuery.replace(/^[^/@]+@(?=[^/]+[:/])/, '');
  }
}

export function readGitSnapshot(cwd, spawn = spawnSync) {
  if (typeof cwd !== 'string' || cwd.length === 0) return null;

  const root = git(cwd, ['rev-parse', '--show-toplevel'], spawn);
  if (!root) return null;

  const branch = git(root, ['branch', '--show-current'], spawn) || 'DETACHED';
  const head = git(root, ['rev-parse', '--short', 'HEAD'], spawn) || 'unborn';
  const porcelain = git(root, ['status', '--porcelain=v1', '--untracked-files=normal'], spawn);
  const trackingOutput = git(root, ['rev-parse', '--abbrev-ref', '--symbolic-full-name', '@{upstream}'], spawn);
  const tracking = trackingOutput || null;
  const trackingCountOutput = tracking
    ? git(root, ['rev-list', '--left-right', '--count', 'HEAD...@{upstream}'], spawn)
    : null;
  const trackingCounts = trackingCountOutput === null ? null : trackingCountOutput.split(/\s+/);
  const originMain = git(root, ['rev-parse', '--verify', 'origin/main'], spawn);
  const baseAheadOutput = originMain
    ? git(root, ['rev-list', '--count', 'origin/main..HEAD'], spawn)
    : null;
  const baseBehindOutput = originMain
    ? git(root, ['rev-list', '--count', 'HEAD..origin/main'], spawn)
    : null;
  const baseAhead = originMain === null || baseAheadOutput === null ? null : parseCount(baseAheadOutput);
  const baseBehind = originMain === null || baseBehindOutput === null ? null : parseCount(baseBehindOutput);
  const inventoryPath = 'docs/fork/diff-inventory.md';
  const inventoryStatus = git(root, ['status', '--porcelain=v1', '--', inventoryPath], spawn);
  const inventoryCommitted = originMain
    ? git(root, ['diff', '--name-only', 'origin/main...HEAD', '--', inventoryPath], spawn)
    : null;

  return {
    root,
    branch,
    head,
    dirty: porcelain === null ? null : Boolean(porcelain),
    tracking,
    ahead: tracking ? (trackingCounts === null ? null : parseCount(trackingCounts?.[0])) : 0,
    behind: tracking ? (trackingCounts === null ? null : parseCount(trackingCounts?.[1])) : 0,
    baseAhead,
    baseBehind,
    originUrl: sanitizeRemoteUrl(git(root, ['remote', 'get-url', 'origin'], spawn)),
    upstreamUrl: sanitizeRemoteUrl(git(root, ['remote', 'get-url', 'upstream'], spawn)),
    inventoryChanged: inventoryStatus === null ? null : Boolean(inventoryStatus || inventoryCommitted),
  };
}

function stageRank(stage) {
  const rank = STAGE_ORDER.indexOf(stage);
  return rank === -1 ? 0 : rank;
}

export function inferStage(snapshot, observedStage = null, observedResultStatus = 'success') {
  if (!snapshot) return 'B';

  if (
    snapshot.dirty === null ||
    snapshot.baseAhead === null ||
    snapshot.baseBehind === null ||
    snapshot.ahead === null ||
    snapshot.behind === null
  ) {
    return 'B';
  }

  let stage = 'B';
  if (snapshot.branch !== 'main' && snapshot.branch !== 'DETACHED') {
    if (snapshot.dirty) stage = 'C1';
    else if (snapshot.baseAhead > 0 && (!snapshot.tracking || snapshot.ahead > 0)) stage = 'C2';
    else if (snapshot.baseAhead > 0 && snapshot.tracking && snapshot.ahead === 0) stage = 'C3';
  }

  // A failed or unknown ship command must never move the workflow forward.
  // The local Git snapshot remains the source of truth for the last completed stage.
  if (observedStage && observedResultStatus === 'success' && stageRank(observedStage) > stageRank(stage)) {
    return observedStage;
  }
  return stage;
}

function commandFromInput(input) {
  const toolInput = input?.tool_input ?? input?.toolInput;
  if (typeof toolInput === 'string') return toolInput;
  if (!toolInput || typeof toolInput !== 'object') return '';
  return typeof toolInput.command === 'string'
    ? toolInput.command
    : typeof toolInput.cmd === 'string'
      ? toolInput.cmd
      : '';
}

function commandResultStatus(input) {
  const response = input?.tool_response ?? input?.toolResponse ?? input?.tool_output ?? input?.toolOutput;
  if (!response || typeof response !== 'object') return 'unknown';
  const exitCode = response.exit_code ?? response.exitCode;
  if (typeof exitCode === 'number') return exitCode === 0 ? 'success' : 'failure';
  if (typeof response.status === 'string') {
    const normalized = response.status.toLowerCase();
    if (['success', 'succeeded', 'passed', 'ok'].includes(normalized)) return 'success';
    if (['failure', 'failed', 'error'].includes(normalized)) return 'failure';
  }
  return 'unknown';
}

function classifyGhApiMutation(command) {
  const apiMatch = command.match(/(?:^|[;&|\r\n]\s*)gh\s+api\s+([^;&|\r\n]*)/);
  if (!apiMatch) return null;

  const args = apiMatch[1].trim().split(/\s+/).filter(Boolean);
  const endpoint = args.find((arg) => /(?:^|\/)repos\/[^\s]+\/pulls(?:\/\d+\/merge)?$/.test(arg));
  if (!endpoint) return null;

  const methodIndex = args.findIndex((arg) => arg === '-X' || arg === '--method');
  const method = methodIndex >= 0 ? args[methodIndex + 1]?.toUpperCase() : null;
  const hasFields = args.some((arg) => ['-f', '--field', '--raw-field', '--input'].includes(arg));
  if (/\/pulls\/\d+\/merge$/.test(endpoint) && method === 'PUT') {
    return { action: 'merge', stage: 'C5' };
  }
  if (/\/pulls$/.test(endpoint) && (method === 'POST' || hasFields)) {
    return { action: 'pr', stage: 'C4' };
  }
  return null;
}

function isReadOnlyShipCommand(command) {
  const push = command.match(/(?:^|[;&|\r\n]\s*)git\s+push\b([^;&|\r\n]*)/);
  if (push) {
    const args = push[1].trim().split(/\s+/).filter(Boolean);
    if (args.includes('--dry-run') || args.includes('-n')) return true;
  }
  if (/(?:^|[;&|\r\n]\s*)git\s+commit\b[^;&|\r\n]*--dry-run/.test(command)) return true;

  const tag = command.match(/(?:^|[;&|\r\n]\s*)git\s+tag(?:\s+([^;&|\r\n]*))?(?:$|[;&|\r\n])/);
  if (tag) {
    const args = (tag[1] ?? '').trim().split(/\s+/).filter(Boolean);
    const readOnlyFlags = new Set([
      '--list',
      '-l',
      '--verify',
      '-v',
      '--contains',
      '--points-at',
      '--merged',
      '--no-merged',
      '--sort',
      '--format',
      '--column',
      '--show-ref',
    ]);
    if (args.length === 0 || args.some((arg) => readOnlyFlags.has(arg) || /^-n\d*$/.test(arg))) return true;
  }
  return false;
}

function isSimpleShipCommand(command) {
  return !/[;&|\r\n`$()]/.test(command.trim());
}

export function classifyShipCommand(command) {
  if (typeof command !== 'string' || command.length === 0) return null;
  if (isReadOnlyShipCommand(command)) return null;
  const apiMutation = classifyGhApiMutation(command);
  if (apiMutation) return apiMutation;
  const separator = String.raw`(^|[;&|\r\n]\s*)`;
  const checks = [
    { action: 'release', stage: 'F', pattern: new RegExp(`${separator}gh\\s+release\\s+create(?:\\s|$)`) },
    {
      action: 'release',
      stage: 'F',
      pattern: new RegExp(
        `${separator}git\\s+push\\b[^;&|\\r\\n]*(?:refs\\/tags\\/|--tags(?:\\s|$)|--follow-tags(?:\\s|$))`,
      ),
    },
    {
      action: 'release',
      stage: 'F',
      pattern: new RegExp(
        `${separator}git\\s+tag(?:\\s+(?!(?:--list|-l|--verify|-v|--contains|--points-at|--merged|--no-merged|--sort|--format|--column|--show-ref|-n\\d*)(?:\\s|$))|$)`,
      ),
    },
    { action: 'merge', stage: 'C5', pattern: new RegExp(`${separator}gh\\s+pr\\s+merge(?:\\s|$)`) },
    { action: 'pr', stage: 'C4', pattern: new RegExp(`${separator}gh\\s+pr\\s+create(?:\\s|$)`) },
    { action: 'push', stage: 'C3', pattern: new RegExp(`${separator}git\\s+push(?:\\s|$)`) },
    { action: 'commit', stage: 'C2', pattern: new RegExp(`${separator}git\\s+commit(?:\\s|$)`) },
    { action: 'review', stage: 'C2-pre', pattern: new RegExp(`${separator}codex\\s+review(?:\\s|$)`) },
  ];
  return checks.find(({ pattern }) => pattern.test(command)) ?? null;
}

function markerPath(input, stateDir) {
  const raw = `${input?.session_id ?? 'unknown'}-${input?.turn_id ?? 'session'}`;
  return join(stateDir, `${raw.replace(/[^a-zA-Z0-9_.-]/g, '_')}.json`);
}

function readMarker(input, stateDir) {
  try {
    return JSON.parse(readFileSync(markerPath(input, stateDir), 'utf8'));
  } catch {
    return null;
  }
}

function writeMarker(input, stateDir, value) {
  try {
    mkdirSync(stateDir, { recursive: true, mode: 0o700 });
    writeFileSync(markerPath(input, stateDir), `${JSON.stringify(value)}\n`, { mode: 0o600 });
  } catch {
    // Hook context must degrade safely when temporary state is unavailable.
  }
}

function additionalContext(event, content) {
  return {
    hookSpecificOutput: {
      hookEventName: event,
      additionalContext: content,
    },
  };
}

function branchStatus(snapshot) {
  const tracking = snapshot.tracking ?? '无 tracking';
  const dirty = snapshot.dirty === null ? 'unknown' : snapshot.dirty ? 'dirty' : 'clean';
  const ahead = snapshot.ahead === null ? 'unknown' : snapshot.ahead;
  const behind = snapshot.behind === null ? 'unknown' : snapshot.behind;
  return `${snapshot.branch} @ ${snapshot.head}; ${dirty}; tracking=${tracking}; ahead=${ahead}; behind=${behind}`;
}

export function formatShipStatus(
  snapshot,
  completedStage = null,
  completedActions = {},
  latestAction = null,
  latestResultStatus = null,
) {
  const stage = inferStage(snapshot, completedStage);
  const failedAction = latestResultStatus === 'failure' ? latestAction : null;
  const unknownAction = latestResultStatus === 'unknown' ? latestAction : null;
  const prCompleted = Boolean(completedActions.pr || completedActions.merge);
  const mergeCompleted = Boolean(completedActions.merge);
  let pushStatus = '未由本地状态确认';
  if (
    snapshot.dirty === null ||
    snapshot.baseAhead === null ||
    snapshot.baseBehind === null ||
    snapshot.ahead === null ||
    snapshot.behind === null
  ) {
    pushStatus = '本地 Git 状态未知，未确认';
  }
  if (failedAction === 'push') {
    pushStatus = '上次 push 失败，仍未完成';
  } else if (unknownAction === 'push') {
    pushStatus = '上次 push 结果未确认';
  } else if (snapshot.tracking && snapshot.ahead > 0 && snapshot.behind > 0) {
    pushStatus = `与 ${snapshot.tracking} 已分叉；领先 ${snapshot.ahead}，落后 ${snapshot.behind}`;
  } else if (snapshot.tracking && snapshot.behind > 0) {
    pushStatus = `未与 ${snapshot.tracking} 对齐；远端领先 ${snapshot.behind}`;
  } else if (snapshot.tracking && snapshot.ahead === 0 && snapshot.behind === 0 && snapshot.baseAhead > 0) {
    pushStatus = `已与 ${snapshot.tracking} 对齐`;
  } else if (snapshot.ahead > 0) {
    pushStatus = `未推送，领先 ${snapshot.ahead}`;
  }
  const next = {
    B: '实现或开 topic 分支；任何 ship 动作仍需当前消息明确授权',
    C1: '先完成验证与 C2-pre；commit 需当前消息明确授权',
    'C2-pre': '审查通过后，commit 仍需当前消息明确授权',
    C2: 'push 需当前消息明确授权',
    C3: '创建 PR 需当前消息明确授权',
    C4: '等待 CI；merge 需当前消息明确授权',
    C5: '清理后如需发版，tag/release 需当前消息明确授权',
    F: '核对 release/镜像结果；部署仍需当前消息明确授权',
  }[stage];

  return [
    `当前阶段: ${stage}`,
    `当前分支: ${snapshot.branch}`,
    `Commit: HEAD ${snapshot.head}；${snapshot.dirty === null ? '工作区状态未知' : snapshot.dirty ? '仍有未提交改动' : '工作区已提交或无改动'}`,
    `Push: ${pushStatus}`,
    `PR: ${failedAction === 'pr' ? '未创建，PR 命令失败' : prCompleted ? '已观察到成功的 PR 阶段命令；以远端状态核验为准' : '未创建或未确认；通常等待 commit + push'}`,
    `Merge: ${failedAction === 'merge' ? '未合并，merge 命令失败' : mergeCompleted ? '已观察到成功的 merge 命令；以远端状态核验为准' : '未执行或未确认'}`,
    `diff-inventory: ${snapshot.inventoryChanged === null ? '状态未知，需重新核验' : snapshot.inventoryChanged ? '本分支或工作区已修改' : '未修改；合入前按永久差异复核'}`,
    '验证: 以本轮实际执行并报告的测试结果为准；Hook 不伪造通过状态',
    `下一步授权: ${next}`,
    ...(failedAction ? [`上次动作: ${failedAction} 失败；阶段保持在最近成功状态`] : []),
    ...(unknownAction ? [`上次动作: ${unknownAction} 结果未确认；阶段保持在最近成功状态`] : []),
  ].join('\n');
}

function sessionContext(snapshot) {
  return [
    'ggapi 工作流守卫已加载。项目 Hook 只提供上下文，不替代 execpolicy、Git Hooks 或用户授权。',
    `本地状态: ${branchStatus(snapshot)}`,
    `当前阶段估计: ${inferStage(snapshot)}（Hook 只读本地状态，C2-pre/C4/C5/F 需结合实际动作核验）`,
    `remote: origin=${snapshot.originUrl ?? 'unknown'}; upstream=${snapshot.upstreamUrl ?? 'unknown'}`,
    `相对 origin/main: ahead=${snapshot.baseAhead}; behind=${snapshot.baseBehind}`,
    '阶段: B → C1 → C2-pre → C2 → C3 → C4 → C5 → F。',
    '授权边界: 只执行用户当前消息明确写出的 commit/push/PR/merge/tag/release 动词；不从历史消息、dirty tree 或“继续”补推权限。',
    `最终回复必须逐项包含: ${SHIP_FIELDS.join('、')}。没有 PR 等结果时也必须写明原因。`,
  ].join('\n');
}

export function handleHook(input, options = {}) {
  const spawn = options.spawn ?? spawnSync;
  const stateDir = options.stateDir ?? join(tmpdir(), 'ggapi-codex-hooks');
  const event = input?.hook_event_name ?? input?.hookEventName;
  if (typeof event !== 'string') return { continue: true };

  const snapshot = readGitSnapshot(input?.cwd, spawn);
  if (!snapshot) return { continue: true };

  if (event === 'SessionStart') {
    return additionalContext(event, sessionContext(snapshot));
  }

  if (event === 'UserPromptSubmit') {
    return additionalContext(
      event,
      '授权提醒：只按用户当前消息中的明确动词决定 commit、push、PR、merge、tag/release 范围。不要从历史消息、当前 Git 状态或“继续”推断新权限；每个未授权动作都停在前一阶段。',
    );
  }

  if (event === 'PreToolUse' || event === 'PostToolUse') {
    const command = commandFromInput(input);
    const classified = classifyShipCommand(command);
    if (!classified) return {};
    const resultStatus =
      event === 'PostToolUse'
        ? isSimpleShipCommand(command)
          ? commandResultStatus(input)
          : 'unknown'
        : 'pending';

    if (event === 'PostToolUse') {
      const previous = readMarker(input, stateDir);
      let completedStage = previous?.completedStage ?? null;
      const completedActions = { ...(previous?.completedActions ?? {}) };
      if (previous?.resultStatus === 'success' && !completedStage) {
        completedStage = previous.stage;
        if (previous.action) completedActions[previous.action] = true;
      }
      if (resultStatus === 'success') {
        completedActions[classified.action] = true;
        if (stageRank(classified.stage) > stageRank(completedStage)) {
          completedStage = classified.stage;
        }
      }
      writeMarker(input, stateDir, {
        completedStage,
        completedActions,
        latestAction: classified.action,
        latestResultStatus: resultStatus,
      });
    }

    const timing = event === 'PreToolUse' ? '即将尝试' : '已观察到';
    return additionalContext(
      event,
      `${timing} ${classified.action} 阶段命令；结果状态=${resultStatus}。当前分支状态: ${branchStatus(snapshot)}。Hook 不代表用户已授权，也不替代实际工具结果；必须据此更新完整 Ship Status。`,
    );
  }

  if (event === 'Stop') {
    if (input.stop_hook_active ?? input.stopHookActive) return { continue: true };
    const marker = readMarker(input, stateDir);
    if (!marker || marker.latestAction === 'review') return { continue: true };

    return {
      decision: 'block',
      reason: [
        '请用下面的完整状态块重新结束本轮；逐项替换成实际结果，不得省略字段。没有执行的动作必须说明原因。',
        formatShipStatus(
          snapshot,
          marker.completedStage,
          marker.completedActions,
          marker.latestAction,
          marker.latestResultStatus,
        ),
      ].join('\n\n'),
    };
  }

  return { continue: true };
}

async function main() {
  let input = {};
  try {
    const chunks = [];
    for await (const chunk of process.stdin) chunks.push(chunk);
    const raw = Buffer.concat(chunks).toString('utf8').trim();
    input = raw ? JSON.parse(raw) : {};
  } catch {
    process.stdout.write('{"continue":true}\n');
    return;
  }

  process.stdout.write(`${JSON.stringify(handleHook(input))}\n`);
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  await main();
}
