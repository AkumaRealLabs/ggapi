import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import {
  classifyShipCommand,
  formatShipStatus,
  handleHook,
  inferStage,
  readGitSnapshot,
  sanitizeRemoteUrl,
} from './ggapi-workflow.mjs';

function run(cwd, ...args) {
  return execFileSync('git', args, { cwd, encoding: 'utf8' }).trim();
}

function createRepo() {
  const root = mkdtempSync(join(tmpdir(), 'ggapi-hook-test-'));
  const remote = join(root, 'origin.git');
  const repo = join(root, 'repo');
  execFileSync('git', ['init', '--bare', remote]);
  execFileSync('git', ['clone', remote, repo]);
  run(repo, 'config', 'user.name', 'Hook Test');
  run(repo, 'config', 'user.email', 'hook@example.com');
  run(repo, 'checkout', '-b', 'main');
  writeFileSync(join(repo, 'README.md'), 'base\n');
  run(repo, 'add', 'README.md');
  run(repo, 'commit', '-m', 'base');
  run(repo, 'push', '-u', 'origin', 'main');
  return { root, remote, repo };
}

test('SessionStart reports clean main using local Git only', () => {
  const { repo } = createRepo();
  const output = handleHook({ hook_event_name: 'SessionStart', cwd: repo });
  const context = output.hookSpecificOutput.additionalContext;
  assert.match(context, /main .*clean/);
  assert.match(context, /B → C1 → C2-pre → C2 → C3 → C4 → C5 → F/);
  assert.match(context, /当前消息/);
});

test('SessionStart redacts credentials and query data from remote URLs', () => {
  const { repo } = createRepo();
  run(
    repo,
    'remote',
    'set-url',
    'origin',
    'https://user:secret@github.com/AkumaRealLabs/ggapi.git?token=private#fragment',
  );
  const output = handleHook({ hook_event_name: 'SessionStart', cwd: repo });
  const context = output.hookSpecificOutput.additionalContext;
  assert.match(context, /https:\/\/github\.com\/AkumaRealLabs\/ggapi\.git/);
  assert.doesNotMatch(context, /user|secret|token=|private|fragment/);
  assert.equal(sanitizeRemoteUrl('git@github.com:AkumaRealLabs/ggapi.git'), 'github.com:AkumaRealLabs/ggapi.git');
  assert.equal(sanitizeRemoteUrl('user:secret@host:path?token=private'), 'host:path');
});

test('snapshot and stage cover topic dirty, ahead, and behind states', () => {
  const { root, remote, repo } = createRepo();
  run(repo, 'checkout', '-b', 'feat/hooks');
  writeFileSync(join(repo, 'topic.txt'), 'dirty\n');
  let snapshot = readGitSnapshot(repo);
  assert.equal(snapshot.dirty, true);
  assert.equal(inferStage(snapshot), 'C1');

  run(repo, 'add', 'topic.txt');
  run(repo, 'commit', '-m', 'topic');
  snapshot = readGitSnapshot(repo);
  assert.equal(snapshot.baseAhead, 1);
  assert.equal(inferStage(snapshot), 'C2');

  const peer = join(root, 'peer');
  execFileSync('git', ['clone', remote, peer]);
  run(peer, 'config', 'user.name', 'Hook Test');
  run(peer, 'config', 'user.email', 'hook@example.com');
  run(peer, 'checkout', 'main');
  writeFileSync(join(peer, 'remote.txt'), 'remote\n');
  run(peer, 'add', 'remote.txt');
  run(peer, 'commit', '-m', 'remote');
  run(peer, 'push', 'origin', 'main');
  run(repo, 'fetch', 'origin');
  snapshot = readGitSnapshot(repo);
  assert.equal(snapshot.baseBehind, 1);
});

test('snapshot includes committed inventory changes and reports tracking divergence', () => {
  const { root, remote, repo } = createRepo();
  run(repo, 'checkout', '-b', 'docs/tracking');
  mkdirSync(join(repo, 'docs/fork'), { recursive: true });
  writeFileSync(join(repo, 'docs/fork/diff-inventory.md'), 'changed\n');
  run(repo, 'add', 'docs/fork/diff-inventory.md');
  run(repo, 'commit', '-m', 'inventory');
  run(repo, 'push', '-u', 'origin', 'docs/tracking');

  let snapshot = readGitSnapshot(repo);
  assert.equal(snapshot.inventoryChanged, true);
  assert.match(formatShipStatus(snapshot), /diff-inventory: 本分支或工作区已修改/);

  const peer = join(root, 'tracking-peer');
  execFileSync('git', ['clone', remote, peer]);
  run(peer, 'config', 'user.name', 'Hook Test');
  run(peer, 'config', 'user.email', 'hook@example.com');
  run(peer, 'checkout', 'docs/tracking');
  writeFileSync(join(peer, 'peer.txt'), 'remote\n');
  run(peer, 'add', 'peer.txt');
  run(peer, 'commit', '-m', 'remote topic');
  run(peer, 'push', 'origin', 'docs/tracking');
  run(repo, 'fetch', 'origin');

  snapshot = readGitSnapshot(repo);
  assert.equal(snapshot.behind, 1);
  assert.match(
    formatShipStatus(snapshot),
    /Push: 未与 origin\/docs\/tracking 对齐；远端领先 1/,
  );
});

test('invalid input and non-Git directories degrade safely', () => {
  const directory = mkdtempSync(join(tmpdir(), 'ggapi-hook-nongit-'));
  assert.deepEqual(handleHook(null), { continue: true });
  assert.deepEqual(handleHook({ hook_event_name: 'Stop', cwd: directory }), { continue: true });
  assert.equal(readGitSnapshot(directory), null);
});

test('UserPromptSubmit does not infer intent from prompt text', () => {
  const { repo } = createRepo();
  const output = handleHook({
    hook_event_name: 'UserPromptSubmit',
    cwd: repo,
    prompt: '继续',
  });
  assert.match(output.hookSpecificOutput.additionalContext, /不要从历史消息/);
  assert.doesNotMatch(output.hookSpecificOutput.additionalContext, /已授权/);
});

test('tool hooks classify ship commands but do not claim hard enforcement', () => {
  assert.deepEqual(classifyShipCommand('git commit -m test')?.stage, 'C2');
  assert.deepEqual(classifyShipCommand('git push origin HEAD')?.stage, 'C3');
  assert.deepEqual(classifyShipCommand('gh pr create --base main')?.stage, 'C4');
  assert.deepEqual(classifyShipCommand('gh pr merge 1')?.stage, 'C5');
  assert.deepEqual(classifyShipCommand('gh api repos/AkumaRealLabs/ggapi/pulls -f title=Test -f head=topic -f base=main')?.stage, 'C4');
  assert.deepEqual(classifyShipCommand('gh api -X PUT repos/AkumaRealLabs/ggapi/pulls/7/merge -f sha=abc')?.stage, 'C5');
  assert.deepEqual(classifyShipCommand('git tag v1')?.stage, 'F');
  assert.deepEqual(classifyShipCommand('git push origin refs/tags/v1:refs/tags/v1')?.stage, 'F');
  assert.deepEqual(classifyShipCommand('gh release create v1')?.stage, 'F');
  assert.deepEqual(
    classifyShipCommand('git tag v1 && git push origin refs/tags/v1:refs/tags/v1')?.stage,
    'F',
  );
  assert.deepEqual(classifyShipCommand('git status\ngit commit -m test')?.stage, 'C2');
  assert.deepEqual(classifyShipCommand('git tag --list')?.stage, undefined);
  assert.equal(classifyShipCommand('git tag --list'), null);
  assert.equal(classifyShipCommand('git tag'), null);
  assert.equal(classifyShipCommand('git tag -n5'), null);
  assert.equal(classifyShipCommand('git commit --dry-run -m test'), null);
  assert.equal(classifyShipCommand('git push --dry-run origin topic'), null);
  assert.equal(classifyShipCommand('git status'), null);
});

test('PostToolUse reports a known command result without granting authorization', () => {
  const { repo } = createRepo();
  const output = handleHook({
    hook_event_name: 'PostToolUse',
    cwd: repo,
    tool_name: 'Bash',
    tool_input: { command: 'git push origin feat/hooks' },
    tool_response: { exit_code: 1 },
  });
  assert.match(output.hookSpecificOutput.additionalContext, /结果状态=failure/);
  assert.match(output.hookSpecificOutput.additionalContext, /不代表用户已授权/);
});

test('failed ship commands do not advance the Stop stage', () => {
  const { repo } = createRepo();
  run(repo, 'checkout', '-b', 'docs/hooks');
  writeFileSync(join(repo, 'guardrails.md'), 'draft\n');
  const stateDir = mkdtempSync(join(tmpdir(), 'ggapi-hook-state-'));
  handleHook({
    hook_event_name: 'PostToolUse',
    cwd: repo,
    session_id: 'failed-session',
    turn_id: 'failed-turn',
    tool_input: { command: 'git push origin HEAD' },
    tool_response: { exit_code: 1 },
  }, { stateDir });

  const output = handleHook({
    hook_event_name: 'Stop',
    cwd: repo,
    session_id: 'failed-session',
    turn_id: 'failed-turn',
    stateDir,
  }, { stateDir });
  assert.equal(output.decision, 'block');
  assert.match(output.reason, /当前阶段: C1/);
  assert.match(output.reason, /Push: 上次 push 失败/);
  assert.match(output.reason, /上次动作: push 失败；阶段保持在最近成功状态/);
});

test('compound ship commands remain unknown even when the shell exits zero', () => {
  const { repo } = createRepo();
  run(repo, 'checkout', '-b', 'docs/hooks');
  writeFileSync(join(repo, 'guardrails.md'), 'draft\n');
  const stateDir = mkdtempSync(join(tmpdir(), 'ggapi-hook-state-'));
  handleHook({
    hook_event_name: 'PostToolUse',
    cwd: repo,
    session_id: 'compound-session',
    turn_id: 'compound-turn',
    tool_input: { command: 'git push origin HEAD || true' },
    tool_response: { exit_code: 0 },
  }, { stateDir });

  const output = handleHook({
    hook_event_name: 'Stop',
    cwd: repo,
    session_id: 'compound-session',
    turn_id: 'compound-turn',
    stateDir,
  }, { stateDir });
  assert.match(output.reason, /Push: 上次 push 结果未确认/);
  assert.match(output.reason, /上次动作: push 结果未确认/);
});

test('unknown Git snapshots fail closed instead of reporting clean', () => {
  const snapshot = {
    branch: 'docs/hooks',
    head: 'abc1234',
    dirty: null,
    tracking: 'origin/docs/hooks',
    ahead: null,
    behind: null,
    baseAhead: null,
    baseBehind: null,
    inventoryChanged: null,
  };
  assert.equal(inferStage(snapshot), 'B');
  const status = formatShipStatus(snapshot);
  assert.match(status, /工作区状态未知/);
  assert.match(status, /Push: 本地 Git 状态未知/);
  assert.match(status, /diff-inventory: 状态未知/);
});

test('a later failure preserves the highest completed ship stage', () => {
  const { repo } = createRepo();
  run(repo, 'checkout', '-b', 'docs/hooks');
  writeFileSync(join(repo, 'guardrails.md'), 'draft\n');
  const stateDir = mkdtempSync(join(tmpdir(), 'ggapi-hook-state-'));
  const base = {
    cwd: repo,
    session_id: 'stage-session',
    turn_id: 'stage-turn',
  };
  handleHook({
    ...base,
    hook_event_name: 'PostToolUse',
    tool_input: { command: 'gh pr create --base main' },
    tool_response: { exit_code: 0 },
  }, { stateDir });
  handleHook({
    ...base,
    hook_event_name: 'PostToolUse',
    tool_input: { command: 'gh pr merge 1' },
    tool_response: { exit_code: 1 },
  }, { stateDir });

  const output = handleHook({ ...base, hook_event_name: 'Stop', stop_hook_active: false }, { stateDir });
  assert.match(output.reason, /当前阶段: C4/);
  assert.match(output.reason, /PR: 已观察到成功的 PR 阶段命令/);
  assert.match(output.reason, /Merge: 未合并，merge 命令失败/);
});

test('Stop does not block a structured review turn', () => {
  const { repo } = createRepo();
  run(repo, 'checkout', '-b', 'docs/review');
  writeFileSync(join(repo, 'guardrails.md'), 'draft\n');
  const stateDir = mkdtempSync(join(tmpdir(), 'ggapi-hook-state-'));
  const base = {
    cwd: repo,
    session_id: 'review-session',
    turn_id: 'review-turn',
  };
  assert.deepEqual(
    handleHook({ ...base, hook_event_name: 'Stop', stop_hook_active: false }, { stateDir }),
    { continue: true },
  );
  handleHook({
    ...base,
    hook_event_name: 'PostToolUse',
    tool_input: { command: 'codex review --uncommitted' },
    tool_response: { exit_code: 0 },
  }, { stateDir });
  assert.deepEqual(
    handleHook({ ...base, hook_event_name: 'Stop', stop_hook_active: false }, { stateDir }),
    { continue: true },
  );
});

test('release completion reaches F without inventing PR or merge success', () => {
  const { repo } = createRepo();
  const stateDir = mkdtempSync(join(tmpdir(), 'ggapi-hook-state-'));
  const base = {
    cwd: repo,
    session_id: 'release-session',
    turn_id: 'release-turn',
  };
  handleHook({
    ...base,
    hook_event_name: 'PostToolUse',
    tool_input: { command: 'gh release create v1' },
    tool_response: { exit_code: 0 },
  }, { stateDir });
  const output = handleHook({ ...base, hook_event_name: 'Stop', stop_hook_active: false }, { stateDir });
  assert.match(output.reason, /当前阶段: F/);
  assert.match(output.reason, /PR: 未创建或未确认/);
  assert.match(output.reason, /Merge: 未执行或未确认/);
});

test('Stop emits every Ship Status field once and ignores transcript format', () => {
  const { repo } = createRepo();
  run(repo, 'checkout', '-b', 'docs/hooks');
  writeFileSync(join(repo, 'guardrails.md'), 'draft\n');
  const stateDir = mkdtempSync(join(tmpdir(), 'ggapi-hook-state-'));
  handleHook({
    hook_event_name: 'PostToolUse',
    cwd: repo,
    session_id: 'session',
    turn_id: 'turn',
    tool_input: { command: 'git commit -m test' },
    tool_response: { exit_code: 0 },
  }, { stateDir });
  const output = handleHook({
    hook_event_name: 'Stop',
    cwd: repo,
    session_id: 'session',
    turn_id: 'turn',
    transcript_path: '/path/that/does/not/exist.jsonl',
    stop_hook_active: false,
  }, { stateDir });
  assert.equal(output.decision, 'block');
  for (const field of [
    '当前阶段:',
    '当前分支:',
    'Commit:',
    'Push:',
    'PR:',
    'Merge:',
    'diff-inventory:',
    '验证:',
    '下一步授权:',
  ]) {
    assert.match(output.reason, new RegExp(field));
  }
  assert.deepEqual(
    handleHook({ hook_event_name: 'Stop', cwd: repo, stop_hook_active: true }),
    { continue: true },
  );
});

test('formatShipStatus remains explicit when gh is unavailable', () => {
  const { repo } = createRepo();
  const status = formatShipStatus(readGitSnapshot(repo));
  assert.match(status, /PR: 未创建或未确认/);
  assert.match(status, /验证: .*Hook 不伪造通过状态/);
});
