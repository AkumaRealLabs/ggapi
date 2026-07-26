import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  describeRuntime,
  detectSurface,
  formatReviewCommand,
  selectReviewStrategy,
} from "./lib/runtime.mjs";
import {
  classifyShipMutations,
  evaluateShipMutation,
  extractShellCommands,
  recordReviewGate,
  reviewGateStatus,
  startsPersistentShellSession,
} from "./lib/review-gate.mjs";

const SCRIPT = fileURLToPath(new URL("./agent-adapter.mjs", import.meta.url));
const PROJECT_ROOT = fileURLToPath(new URL("../../../../", import.meta.url));

function git(root, ...args) {
  const result = spawnSync("git", args, { cwd: root, encoding: "utf8" });
  assert.equal(result.status, 0, result.stderr || result.stdout);
  return result.stdout.trim();
}

function repositoryFixture() {
  // realpath: on macOS os.tmpdir() is a symlink, and gitInvocationViolation
  // compares against the resolved repository root.
  const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "ggapi-agent-adapter-test-")));
  git(root, "init", "--initial-branch=main");
  git(root, "config", "user.name", "Hook Test");
  git(root, "config", "user.email", "hook-test@example.invalid");
  fs.writeFileSync(path.join(root, ".gitignore"), ".ggapi-agent/\n");
  fs.writeFileSync(path.join(root, "tracked.txt"), "base\n");
  git(root, "add", ".gitignore", "tracked.txt");
  git(root, "commit", "-m", "base");
  git(root, "update-ref", "refs/remotes/origin/main", "HEAD");
  git(root, "switch", "-c", "feat/hook-test");
  fs.writeFileSync(path.join(root, "tracked.txt"), "changed\n");
  return root;
}

function runHook(root, input, surface = "codex", env = {}) {
  return spawnSync(process.execPath, [SCRIPT, "hook", "--surface", surface], {
    cwd: root,
    env: { ...process.env, ...env },
    input: JSON.stringify(input),
    encoding: "utf8",
  });
}

function fakeCommandPath(...commands) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "ggapi-agent-bin-"));
  for (const command of commands) {
    const executable = path.join(root, command);
    fs.writeFileSync(executable, "#!/bin/sh\nexit 0\n", { mode: 0o755 });
  }
  return {
    cleanup: () => fs.rmSync(root, { force: true, recursive: true }),
    path: `${root}${path.delimiter}/usr/bin:/bin`,
    root,
  };
}

test("detects the active agent surface before installed CLI fallbacks", () => {
  assert.equal(detectSurface({ env: { GROK_HOOK_EVENT: "session_start" } }), "grok");
  assert.equal(detectSurface({ env: { CODEX_THREAD_ID: "thread" } }), "codex");
  assert.equal(detectSurface({ env: { CLAUDECODE: "1" } }), "claude");
  assert.equal(detectSurface({ explicit: "generic", env: {} }), "generic");
});

test("selects a Codex-native review command in a Codex session", () => {
  const fake = fakeCommandPath("codex");
  try {
    const runtime = describeRuntime({
      explicit: "codex",
      env: { PATH: fake.path },
      root: process.cwd(),
    });
    assert.equal(runtime.surface, "codex");
    assert.equal(runtime.review.kind, "codex-native");
    assert.equal(formatReviewCommand(runtime.review, "origin/main"), "codex review --base origin/main");
  } finally {
    fake.cleanup();
  }
});

test("reports review unavailable only for an agent with no reviewer of its own", () => {
  // A generic agent has no in-session reviewer to fall back on.
  const strategy = selectReviewStrategy("generic", { env: { PATH: "" } });
  assert.equal(strategy.kind, "unavailable");
  assert.equal(strategy.reviewer, null);
  assert.equal(formatReviewCommand(strategy), null);
});

test("never substitutes another vendor's reviewer for a surface that has its own", () => {
  const fake = fakeCommandPath("claude", "grok", "codex");
  try {
    // A rate-limited or failing own reviewer is a blocked gate, not a reason to
    // silently review with a different vendor.
    const claude = selectReviewStrategy("claude", { env: { PATH: fake.path } });
    assert.equal(claude.kind, "claude-native");
    assert.equal(claude.reviewer, "claude");
    assert.equal(claude.fallback, null);
    assert.equal(formatReviewCommand(claude, "origin/main"), "claude -p '/code-review origin/main'");

    const grok = selectReviewStrategy("grok", { env: { PATH: fake.path } });
    assert.equal(grok.kind, "grok-native");
    assert.equal(grok.reviewer, "grok");
    assert.equal(grok.fallback, null);
    const command = formatReviewCommand(grok, "origin/release");
    assert.match(command, /^grok -p '/);
    assert.match(command, /origin\/release/);
    assert.doesNotMatch(command, /origin\/main|codex/);
  } finally {
    fake.cleanup();
  }
});

test("keeps Claude on its own reviewer even without a claude CLI", () => {
  const fake = fakeCommandPath("codex");
  try {
    const strategy = selectReviewStrategy("claude", { env: { PATH: fake.path } });
    assert.equal(strategy.kind, "claude-in-session");
    assert.equal(strategy.reviewer, "claude");
    assert.equal(formatReviewCommand(strategy, "origin/main"), null);
  } finally {
    fake.cleanup();
  }
});

test("uses the Codex CLI only for a generic agent with no reviewer of its own", () => {
  const fake = fakeCommandPath("codex");
  try {
    const strategy = selectReviewStrategy("generic", { env: { PATH: fake.path } });
    assert.equal(strategy.kind, "codex-cli-fallback");
    assert.equal(strategy.reviewer, "codex");
    assert.equal(formatReviewCommand(strategy, "origin/main"), "codex review --base origin/main");
  } finally {
    fake.cleanup();
  }
});

test("classifies ship mutations without blocking read-only command examples", () => {
  const direct = classifyShipMutations(["git commit -m 'feat: test' && git push -u origin HEAD"]);
  assert.deepEqual(direct.map(({ kind }) => kind), ["commit", "push"]);

  const functionInput = `const result = await tools.exec_command({cmd:"git commit -m 'fix: test'"});`;
  const extracted = extractShellCommands("functions.exec", functionInput);
  assert.deepEqual(classifyShipMutations(extracted).map(({ kind }) => kind), ["commit"]);
  assert.deepEqual(
    classifyShipMutations(extractShellCommands("exec", functionInput)).map(({ kind }) => kind),
    ["commit"],
  );
  assert.deepEqual(
    classifyShipMutations(extractShellCommands("functions.exec", { code: functionInput })).map(({ kind }) => kind),
    ["commit"],
  );

  assert.deepEqual(classifyShipMutations([
    "rg -n 'git commit' docs",
    "git tag --list",
    "git tag -l 'v*'",
    "git commit --dry-run",
    "git push --dry-run origin HEAD",
  ]), []);

  assert.deepEqual(
    classifyShipMutations(["git -C . commit -m 'fix: test'", "git -c advice.detachedHead=false push origin HEAD"])
      .map(({ kind }) => kind),
    ["commit", "push"],
  );

  const assembled = `const cmd = "git commit -m 'fix: assembled'"; await tools.exec_command({cmd});`;
  assert.deepEqual(classifyShipMutations(extractShellCommands("functions.exec", assembled)).map(({ kind }) => kind), ["commit"]);

  const dynamic = `const subcommand = "push"; await tools.exec_command({cmd: \`git \${subcommand} origin HEAD\`});`;
  assert.deepEqual(classifyShipMutations(extractShellCommands("exec", dynamic)).map(({ kind }) => kind), ["unknown"]);

  const aliasedTool = `const run = tools.exec_command; await run({cmd: "git push origin HEAD"});`;
  assert.deepEqual(classifyShipMutations(extractShellCommands("exec", aliasedTool)).map(({ kind }) => kind), ["unknown"]);

  const stdin = `await tools.write_stdin({session_id: 42, chars: "git push origin HEAD\\n"});`;
  assert.deepEqual(classifyShipMutations(extractShellCommands("exec", stdin)).map(({ kind }) => kind), ["push"]);

  const multiCall = `
    await tools.exec_command({cmd: "sed -i s/a/b/ tracked.txt"});
    await tools.exec_command({cmd: "git commit -am 'fix: composite'"});
  `;
  assert.deepEqual(classifyShipMutations(extractShellCommands("functions.exec", multiCall)).map(({ kind }) => kind), ["commit", "unknown"]);

  const editThenCommit = `
    await tools.apply_patch("*** Begin Patch\\n*** End Patch");
    await tools.exec_command({cmd: "git commit -am 'fix: composite edit'"});
  `;
  assert.deepEqual(
    classifyShipMutations(extractShellCommands("functions.exec", editThenCommit)).map(({ kind }) => kind),
    ["commit", "unknown"],
  );
});

test("rejects composite ship commands and non-exact temporary subjects", () => {
  const composite = classifyShipMutations(["sed -i s/a/b/ tracked.txt && git commit -am 'fix: changed'"])[0];
  assert.equal(composite.compound, true);

  const disguised = classifyShipMutations([
    "git commit -m 'fix: real subject' # chore: temp codex gate snapshot",
  ])[0];
  assert.equal(disguised.temporarySnapshot, false);

  const wrapped = classifyShipMutations(["sh -lc \"sed -i s/a/b/ tracked.txt && git commit -am 'fix: wrapped'\""])[0];
  assert.equal(wrapped.kind, "commit");
  assert.equal(wrapped.compound, true);

  for (const nested of [
    "if git push origin HEAD; then echo ok; fi",
    "echo $(git push origin HEAD)",
    "(git commit -m 'fix: nested')",
    "find . -exec git tag v1.0.0 \\;",
    "cmd='git push origin HEAD'; bash -lc \"$cmd\"",
  ]) {
    assert.deepEqual(classifyShipMutations([nested]).map(({ kind }) => kind), ["unknown"]);
  }
});

test("denies ship mutations smuggled alongside an auto-allowed temp snapshot", () => {
  // "&" backgrounds the commit and runs the push; both must be denied even
  // though the visible commit is the allowed materialization message.
  const backgrounded = classifyShipMutations([
    "git commit -m 'chore: temp codex gate snapshot' & git push origin HEAD:refs/heads/main",
  ]);
  assert.deepEqual(backgrounded.map(({ kind }) => kind), ["commit", "push"]);
  assert.ok(backgrounded.every(({ compound }) => compound));

  // A recognized action must not suppress the nested-mutation scan.
  const substituted = classifyShipMutations([
    "git commit -m 'chore: temp codex gate snapshot' $(git push origin main)",
  ]);
  assert.equal(substituted[0].kind, "commit");
  assert.equal(substituted[0].compound, true);

  // Descriptor redirection keeps its "&" and stays a single clean action.
  const redirected = classifyShipMutations(["git push origin HEAD 2>&1"]);
  assert.deepEqual(redirected.map(({ compound, kind }) => ({ compound, kind })), [{ compound: false, kind: "push" }]);
});

test("classifies nested shells beyond an exact bash -c", () => {
  for (const command of [
    'bash -ec "git push origin main"',
    'dash -c "git push origin main"',
    'ksh -c "git push origin main"',
  ]) {
    assert.deepEqual(classifyShipMutations([command]).map(({ kind }) => kind), ["push"], command);
  }

  // Shell payload we cannot read must fail closed, not silently allow.
  assert.deepEqual(
    classifyShipMutations(["bash ship-it.sh git push"]).map(({ compound, kind }) => ({ compound, kind })),
    [{ compound: true, kind: "unknown" }],
  );
  assert.deepEqual(classifyShipMutations(["bash build.sh"]), []);
});

test("inspects argv-array and stdin shell payloads instead of failing open", () => {
  assert.deepEqual(
    classifyShipMutations(extractShellCommands("Bash", { command: ["git", "push", "origin", "main"] }))
      .map(({ kind }) => kind),
    ["push"],
  );
  assert.deepEqual(
    classifyShipMutations(extractShellCommands("write_stdin", { chars: "git push origin HEAD\n" }))
      .map(({ kind }) => kind),
    ["push"],
  );
  // An unreadable payload shape must not classify as "nothing to gate".
  assert.deepEqual(
    classifyShipMutations(extractShellCommands("Bash", { command: { argv: ["git", "push"] } }))
      .map(({ kind }) => kind),
    ["unknown"],
  );
});

test("joins adjacent quote fragments into one shell word", () => {
  // Every one of these runs a real commit/push in a shell.
  for (const command of ['git ""commit -m x', 'git c"o"mmit -m x', "git 'pu'sh origin HEAD", 'git pu"sh" origin HEAD']) {
    assert.equal(classifyShipMutations([command]).length, 1, command);
  }
  assert.deepEqual(
    classifyShipMutations(['git "push" origin HEAD']).map(({ compound, kind }) => ({ compound, kind })),
    [{ compound: false, kind: "push" }],
  );
});

test("detects grouped and substituted remote-ref mutations", () => {
  // tokensDescribeShipMutation is the only detector for these, so it must know
  // every command classifyShipMutations knows.
  for (const command of [
    "(gh release create v9.9.9)",
    "git status && $(gh release create v9.9.9)",
    "EP=repos/AkumaRealLabs/ggapi/pulls/12/merge; gh api -X PUT $EP",
  ]) {
    assert.deepEqual(classifyShipMutations([command]).map(({ kind }) => kind), ["unknown"], command);
  }
});

test("treats duplicate command keys as unresolved", () => {
  // JavaScript keeps the last duplicate; reading the first would classify the
  // benign one and run the other.
  const payload = `tools.${"ex" + "ec_command"}({cmd: "echo ok", cmd: "git " + "push origin main"})`;
  assert.deepEqual(
    classifyShipMutations(extractShellCommands("exec", payload)).map(({ kind }) => kind),
    ["unknown"],
  );
});

test("preserves argv quoting so nested shell payloads stay visible", () => {
  // Joining argv without quoting would leave "-lc" holding the bare token
  // "git" and hide the push entirely.
  const argv = extractShellCommands("Bash", { command: ["bash", "-lc", "git push origin feature"] });
  assert.deepEqual(classifyShipMutations(argv).map(({ kind }) => kind), ["push"]);
});

test("treats an unresolved command expression as uninspectable", () => {
  for (const payload of [
    `tools.exec_command({cmd: "git" + " push origin HEAD"})`,
    'tools.exec_command({cmd: `git ${subcommand} origin HEAD`})',
  ]) {
    assert.deepEqual(
      classifyShipMutations(extractShellCommands("exec", payload)).map(({ kind }) => kind),
      ["unknown"],
      payload,
    );
  }
});

test("refuses a release tag whose target cannot be resolved", () => {
  const root = repositoryFixture();
  try {
    git(root, "restore", "tracked.txt");
    git(root, "switch", "main");
    // Falling back to "assume origin/main" would make the tip rule vacuous.
    const action = classifyShipMutations(["gh release delete v1.0.0-rc.20.1"])[0];
    const decision = evaluateShipMutation(root, action);
    assert.equal(decision.allowed, false);
    assert.match(decision.reason, /target could not be resolved/);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("gates remote ref creation that bypasses the git tag path", () => {
  assert.deepEqual(
    classifyShipMutations(["gh release create v1.0.0-rc.20.1 --target abc1234"]).map(({ kind }) => kind),
    ["tag"],
  );
  assert.deepEqual(
    classifyShipMutations(["gh api -X POST repos/AkumaRealLabs/ggapi/git/refs -f ref=refs/tags/v1 -f sha=abc1234"])
      .map(({ kind }) => kind),
    ["tag"],
  );
  assert.deepEqual(classifyShipMutations(["gh release view v1.0.0-rc.20.1"]), []);
});

test("denies a commit whose index holds content the reviewed worktree lacks", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: materialize reviewed tree");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    const commit = classifyShipMutations(["git commit -m 'feat: land reviewed tree'"])[0];
    assert.equal(evaluateShipMutation(root, commit).allowed, true);

    // Stage content, then remove it from the worktree: the worktree snapshot
    // matches the gate again while the index would commit something unreviewed.
    fs.writeFileSync(path.join(root, "smuggled.txt"), "never reviewed\n");
    git(root, "add", "smuggled.txt");
    fs.rmSync(path.join(root, "smuggled.txt"));
    const decision = evaluateShipMutation(root, commit);
    assert.equal(decision.allowed, false);
    assert.match(decision.reason, /staged index does not match/);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("checks the staged index even when the worktree matches the base tree", () => {
  const root = repositoryFixture();
  try {
    git(root, "restore", "tracked.txt");
    // Worktree tree == origin/main tree, so the gate state is "empty" and the
    // index is the only thing carrying content into the commit.
    fs.writeFileSync(path.join(root, "smuggled.txt"), "unreviewed\n");
    git(root, "add", "smuggled.txt");
    fs.rmSync(path.join(root, "smuggled.txt"));
    const decision = evaluateShipMutation(root, classifyShipMutations(["git commit -m 'feat: x'"])[0]);
    assert.equal(decision.allowed, false);
    assert.match(decision.reason, /staged index does not match/);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("rejects redirecting git environment and bulk tag pushes", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    // commandTokens strips the VAR=value prefix, so these must be re-attached
    // to the invocation guard rather than silently dropped.
    for (const command of ["GIT_DIR=/other/.git git push origin HEAD", "GIT_WORK_TREE=/other git commit -m 'x'"]) {
      const decision = evaluateShipMutation(root, classifyShipMutations([command])[0]);
      assert.equal(decision.allowed, false, command);
      assert.match(decision.reason, /override Git configuration or repository/);
    }

    for (const command of ["git push --follow-tags origin HEAD", "git push --tags origin HEAD"]) {
      const decision = evaluateShipMutation(root, classifyShipMutations([command])[0]);
      assert.equal(decision.allowed, false, command);
      assert.match(decision.reason, /exact tag refspec/);
    }
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("allows local tag deletion, which touches no remote ref", () => {
  const root = repositoryFixture();
  try {
    git(root, "restore", "tracked.txt");
    // SOP §5 documents this cleanup step; only remote deletion is gated.
    const decision = evaluateShipMutation(root, classifyShipMutations(["git tag -d v1.0.0-rc.20.1"])[0]);
    assert.equal(decision.allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("checks a tool payload whose event name is missing or misspelled", () => {
  const root = repositoryFixture();
  try {
    const denied = runHook(root, {
      cwd: root,
      tool_input: { command: "git commit -m 'feat: no event name'" },
      tool_name: "Bash",
    }, "claude");
    assert.equal(denied.status, 0, denied.stderr);
    assert.equal(JSON.parse(denied.stdout).hookSpecificOutput.permissionDecision, "deny");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("reads glued short options and flags used as option values", () => {
  // -XPUT is -X PUT; missing it made an unreviewed merge look like a GET.
  assert.deepEqual(
    classifyShipMutations(["gh api -XPUT repos/o/r/pulls/12/merge -f sha=abc1234"]).map(({ kind }) => kind),
    ["merge-rest"],
  );
  assert.deepEqual(
    classifyShipMutations(["gh api -XDELETE repos/o/r/git/refs/tags/v1"]).map(({ kind }) => kind),
    ["tag"],
  );
  // "--dry-run" as the VALUE of -m is a real commit, not a dry run.
  assert.deepEqual(classifyShipMutations(["git commit -m --dry-run"]).map(({ kind }) => kind), ["commit"]);
  assert.deepEqual(classifyShipMutations(["git tag -m --help v1.0.0"]).map(({ kind }) => kind), ["tag"]);
  assert.deepEqual(classifyShipMutations(["git commit --dry-run"]), []);
});

test("separates remote tag deletion from ordinary branch deletion", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    const tag = evaluateShipMutation(root, classifyShipMutations(["git push origin --delete refs/tags/v1.0.0"])[0]);
    assert.equal(tag.allowed, false);
    assert.match(tag.reason, /Remote tag deletion/);

    const branch = evaluateShipMutation(root, classifyShipMutations(["git push origin --delete refs/heads/feat/old"])[0]);
    assert.equal(branch.allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("accepts a bundled -am commit and rejects config-redirecting environment", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    // -a is bundled here, so the index comparison must not apply.
    assert.equal(evaluateShipMutation(root, classifyShipMutations(["git commit -am 'msg'"])[0]).allowed, true);

    const redirected = evaluateShipMutation(root, classifyShipMutations(["GIT_CONFIG_COUNT=1 git push origin HEAD"])[0]);
    assert.equal(redirected.allowed, false);
    assert.match(redirected.reason, /override Git configuration or repository/);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("parses command substitution bodies that quoting keeps in one segment", () => {
  // Quoting stops splitShellSegments from separating these, so the body has to
  // be parsed as its own command rather than word-scanned.
  for (const command of [
    'git commit -m "$(true; git push origin HEAD:refs/heads/main)"',
    'git commit -m "$(env git push origin HEAD)"',
    'git commit -m "`git push origin HEAD`"',
  ]) {
    const actions = classifyShipMutations([command]);
    assert.equal(actions[0].kind, "commit", command);
    assert.equal(actions[0].compound, true, command);
  }
  // A substitution with no ship mutation stays a plain single action.
  assert.equal(classifyShipMutations(['git commit -m "$(date)"'])[0].compound, false);
});

test("treats a bare tag name as a remote tag deletion", () => {
  const root = repositoryFixture();
  try {
    git(root, "restore", "tracked.txt");
    git(root, "tag", "v1.0.0-rc.21.1");
    git(root, "switch", "-c", "feat/deletes");
    fs.writeFileSync(path.join(root, "tracked.txt"), "changed\n");
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    // "v1.0.0-rc.21.1" deletes the tag exactly as refs/tags/... would.
    const tag = evaluateShipMutation(root, classifyShipMutations(["git push origin --delete v1.0.0-rc.21.1"])[0]);
    assert.equal(tag.allowed, false);
    assert.match(tag.reason, /Remote tag deletion/);

    const branch = evaluateShipMutation(root, classifyShipMutations(["git push origin --delete refs/heads/feat/old"])[0]);
    assert.equal(branch.allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("does not let a flag in value position stand in for -a or -d", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });
    fs.writeFileSync(path.join(root, "smuggled.txt"), "unreviewed\n");
    git(root, "add", "smuggled.txt");
    fs.rmSync(path.join(root, "smuggled.txt"));

    // Here -a is the MESSAGE, so the index comparison must still run.
    const commit = evaluateShipMutation(root, classifyShipMutations(["git commit -m -a"])[0]);
    assert.equal(commit.allowed, false);
    assert.match(commit.reason, /staged index does not match/);

    // Here -d is the MESSAGE, so this creates a tag rather than deleting one.
    const tag = evaluateShipMutation(root, classifyShipMutations(["git tag -m -d v9"])[0]);
    assert.equal(tag.allowed, false);
    assert.match(tag.reason, /origin\/main tip/);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("lets every surface review in-session when its CLI is missing", () => {
  for (const surface of ["claude", "grok", "codex"]) {
    const strategy = selectReviewStrategy(surface, { env: { PATH: "" } });
    assert.equal(strategy.kind, `${surface}-in-session`, surface);
    assert.equal(strategy.reviewer, surface, surface);
  }
});

test("computes the gh api verb the way gh does", () => {
  // gh api switches to POST as soon as a field is sent, so the absence of -X
  // does not mean the call is a read.
  assert.deepEqual(
    classifyShipMutations(["gh api repos/o/r/git/refs -f ref=refs/tags/v9.9.9 -f sha=abc1234"]).map(({ kind }) => kind),
    ["tag"],
  );
  assert.deepEqual(classifyShipMutations(["gh api repos/o/r/git/refs"]), []);
});

test("recognizes every spelling of the main branch destination", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    // git resolves "heads/main" to refs/heads/main just as it does "main".
    for (const refspec of ["HEAD:heads/main", "HEAD:refs/heads/main", "main"]) {
      const decision = evaluateShipMutation(root, classifyShipMutations([`git push origin ${refspec}`])[0]);
      assert.equal(decision.allowed, false, refspec);
      assert.match(decision.reason, /directly to main/, refspec);
    }
    assert.equal(evaluateShipMutation(root, classifyShipMutations(["git push origin HEAD"])[0]).allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("recognizes a tag name that contains slashes", () => {
  const root = repositoryFixture();
  try {
    git(root, "restore", "tracked.txt");
    git(root, "tag", "release/2026-07");
    git(root, "switch", "-c", "feat/slashed");
    fs.writeFileSync(path.join(root, "tracked.txt"), "changed\n");
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    const tag = evaluateShipMutation(root, classifyShipMutations(["git push origin --delete release/2026-07"])[0]);
    assert.equal(tag.allowed, false);
    assert.match(tag.reason, /Remote tag deletion/);

    const branch = evaluateShipMutation(root, classifyShipMutations(["git push origin --delete refs/heads/feat/old"])[0]);
    assert.equal(branch.allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("validates the repository the tool call actually runs in", () => {
  const root = repositoryFixture();
  const other = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    // `other` carries a committed but unreviewed tip; naming it as the call's
    // workdir must not borrow this repository's clean marker.
    git(other, "add", "tracked.txt");
    git(other, "commit", "-m", "feat: never reviewed");
    const denied = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      tool_input: { command: "git push origin HEAD", workdir: other },
      tool_name: "Bash",
    }, "claude");
    assert.equal(denied.status, 0, denied.stderr);
    assert.equal(JSON.parse(denied.stdout).hookSpecificOutput.permissionDecision, "deny");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
    fs.rmSync(other, { force: true, recursive: true });
  }
});

test("resolves a constant only when it is the whole command", () => {
  const method = "ex" + "ec_command";
  const concatenated = `const C = "echo ok"; tools.${method}({cmd: C + "; git push origin main"})`;
  assert.deepEqual(
    classifyShipMutations(extractShellCommands("exec", concatenated)).map(({ kind }) => kind),
    ["unknown"],
  );
  const plain = `const C = "echo ok"; tools.${method}({cmd: C})`;
  assert.deepEqual(classifyShipMutations(extractShellCommands("exec", plain)), []);
});

test("applies the tag tip rule to a bare tag name push", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    git(root, "tag", "v9.9.9");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    // "v9.9.9" publishes the tag exactly as refs/tags/v9.9.9 would.
    const tag = evaluateShipMutation(root, classifyShipMutations(["git push origin v9.9.9"])[0]);
    assert.equal(tag.allowed, false);
    assert.match(tag.reason, /origin\/main tip/);

    assert.equal(evaluateShipMutation(root, classifyShipMutations(["git push origin HEAD"])[0]).allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("does not deny read-only calls that name a workdir outside any repository", () => {
  const root = repositoryFixture();
  const outside = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "ggapi-not-a-repo-")));
  try {
    const allowed = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      tool_input: { command: "ls -la", workdir: outside },
      tool_name: "Bash",
    }, "claude");
    assert.equal(allowed.status, 0, allowed.stderr);
    assert.equal(allowed.stdout, "");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
    fs.rmSync(outside, { force: true, recursive: true });
  }
});

test("treats a ship mutation inside an interpreter argument as uninspectable", () => {
  // These are not shell, so the payload cannot be tokenized — but ignoring it
  // would let the mutation run.
  for (const command of [
    `node -e "require('child_process').execSync('git push origin HEAD')"`,
    `python3 -c "import os; os.system('git push origin HEAD')"`,
  ]) {
    assert.deepEqual(classifyShipMutations([command]).map(({ kind }) => kind), ["unknown"], command);
  }
  assert.deepEqual(classifyShipMutations([`node -e "console.log(1)"`]), []);
  assert.deepEqual(classifyShipMutations(["node --version"]), []);
});

test("keeps the environment guard when the mutation runs in a nested shell", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    const wrapped = evaluateShipMutation(root, classifyShipMutations(["GIT_CONFIG_COUNT=1 sh -c 'git push origin HEAD'"])[0]);
    assert.equal(wrapped.allowed, false);
    assert.match(wrapped.reason, /override Git configuration or repository/);

    assert.equal(evaluateShipMutation(root, classifyShipMutations(["sh -c 'git push origin HEAD'"])[0]).allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("requires an unambiguous ref namespace when deleting", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });

    // A published tag cleaned up locally is indistinguishable from a branch by
    // name, so the caller has to say which namespace they mean.
    const ambiguous = evaluateShipMutation(root, classifyShipMutations(["git push origin --delete v1.0.0"])[0]);
    assert.equal(ambiguous.allowed, false);
    assert.match(ambiguous.reason, /refs\/heads\/v1\.0\.0 or refs\/tags\/v1\.0\.0/);

    assert.equal(
      evaluateShipMutation(root, classifyShipMutations(["git push origin --delete refs/heads/feat/old"])[0]).allowed,
      true,
    );
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("gates gh api release and graphql mutation surfaces", () => {
  assert.deepEqual(
    classifyShipMutations(["gh api repos/o/r/releases -f tag_name=v9.9.9"]).map(({ kind }) => kind),
    ["tag"],
  );
  assert.deepEqual(
    classifyShipMutations(["gh api graphql -f query='mutation { mergePullRequest(input:{pullRequestId:1}) { clientMutationId } }'"])
      .map(({ kind }) => kind),
    ["unknown"],
  );
  assert.deepEqual(classifyShipMutations(["gh api repos/o/r/releases"]), []);
});

test("detects a persistent shell opened in any segment of a command", () => {
  assert.equal(startsPersistentShellSession(["true; bash"]), true);
  assert.equal(startsPersistentShellSession(["git status --short && zsh"]), true);
  assert.equal(startsPersistentShellSession(["zsh -lc 'git status --short'"]), false);
  // These print and exit rather than opening a session.
  assert.equal(startsPersistentShellSession(["bash --version"]), false);
  assert.equal(startsPersistentShellSession(["sh --help"]), false);
});

test("records a clean review when the marker directory is not gitignored", () => {
  const root = repositoryFixture();
  try {
    // A branch or worktree predating the .gitignore entry must still be able to
    // record and keep a clean gate instead of being permanently stale.
    fs.writeFileSync(path.join(root, ".gitignore"), "");
    git(root, "add", ".gitignore", "tracked.txt");
    git(root, "commit", "-m", "feat: materialize reviewed tree without marker ignore");
    recordReviewGate(root, { reviewer: "claude", state: "clean", surface: "claude" });
    assert.equal(reviewGateStatus(root).state, "clean");

    // The marker itself must not count as content that staled the tree.
    assert.ok(fs.existsSync(path.join(root, ".ggapi-agent", "review-gate.json")));
    assert.equal(reviewGateStatus(root).state, "clean");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("keeps a clean review valid across a same-tree commit and invalidates content changes", () => {
  const root = repositoryFixture();
  try {
    assert.equal(reviewGateStatus(root).state, "missing");
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: materialize reviewed tree");
    recordReviewGate(root, { reviewer: "codex", state: "clean", surface: "codex" });
    assert.equal(reviewGateStatus(root).state, "clean");

    git(root, "commit", "--allow-empty", "-m", "feat: preserve reviewed tree");
    assert.equal(reviewGateStatus(root).state, "clean");

    fs.writeFileSync(path.join(root, "tracked.txt"), "changed again\n");
    assert.equal(reviewGateStatus(root).state, "stale");
    assert.throws(
      () => recordReviewGate(root, { state: "bypassed", surface: "codex" }),
      /explicit user-approved reason/,
    );
    recordReviewGate(root, {
      reason: "user explicitly requested skip review",
      state: "bypassed",
      surface: "codex",
    });
    assert.equal(reviewGateStatus(root).state, "bypassed");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("invalidates a clean review when origin/main advances", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: materialize reviewed tree");
    recordReviewGate(root, { reviewer: "codex", state: "clean", surface: "codex" });
    const originalMain = git(root, "rev-parse", "origin/main");
    git(root, "switch", "main");
    fs.writeFileSync(path.join(root, "tracked.txt"), "main moved\n");
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "main moved");
    git(root, "update-ref", "refs/remotes/origin/main", "HEAD", originalMain);
    git(root, "switch", "feat/hook-test");
    assert.equal(reviewGateStatus(root).state, "stale");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("Codex, Grok, and Claude hooks deny an unreviewed final commit with native output", () => {
  const root = repositoryFixture();
  try {
    const codexInput = {
      cwd: root,
      hook_event_name: "PreToolUse",
      model: "test-model",
      tool_input: { cmd: "git commit -m 'feat: test'" },
      tool_name: "exec_command",
      turn_id: "turn",
    };
    const codex = runHook(root, codexInput);
    assert.equal(codex.status, 0, codex.stderr);
    assert.equal(JSON.parse(codex.stdout).hookSpecificOutput.permissionDecision, "deny");

    const codexExec = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      tool_input: "await tools.exec_command({cmd: \"git commit -m 'feat: exec payload'\"})",
      tool_name: "exec",
      turn_id: "turn",
    });
    assert.equal(codexExec.status, 0, codexExec.stderr);
    assert.equal(JSON.parse(codexExec.stdout).hookSpecificOutput.permissionDecision, "deny");

    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: unreviewed push");

    const grokInput = {
      cwd: root,
      hookEventName: "pre_tool_use",
      toolInput: { command: "git push -u origin HEAD" },
      toolName: "run_terminal_command",
    };
    const grok = runHook(root, grokInput, "auto", { GROK_HOOK_EVENT: "pre_tool_use" });
    assert.equal(grok.status, 0, grok.stderr);
    assert.equal(JSON.parse(grok.stdout).decision, "deny");

    fs.writeFileSync(path.join(root, "tracked.txt"), "changed again\n");

    const claude = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      tool_input: { command: "git commit -m 'feat: test'" },
      tool_name: "Bash",
    }, "claude");
    assert.equal(claude.status, 0, claude.stderr);
    assert.equal(JSON.parse(claude.stdout).hookSpecificOutput.permissionDecision, "deny");

    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed final tree");
    recordReviewGate(root, { reviewer: "codex", state: "clean", surface: "codex" });
    const allowed = runHook(root, codexInput);
    assert.equal(allowed.status, 0, allowed.stderr);
    assert.equal(allowed.stdout, "");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("denies truncated shell payloads before classifying their visible prefix", () => {
  const root = repositoryFixture();
  try {
    const grok = runHook(root, {
      cwd: root,
      hookEventName: "pre_tool_use",
      toolInput: { command: "printf visible" },
      toolInputTruncated: true,
      toolName: "run_terminal_command",
    }, "grok");
    assert.equal(grok.status, 0, grok.stderr);
    assert.equal(JSON.parse(grok.stdout).decision, "deny");

    const codex = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      tool_input: "await tools.exec_command({cmd: \"printf visible\"})",
      tool_input_truncated: true,
      tool_name: "exec",
    });
    assert.equal(codex.status, 0, codex.stderr);
    assert.match(JSON.parse(codex.stdout).hookSpecificOutput.permissionDecisionReason, /truncated/);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("denies persistent shells whose later stdin would skip PreToolUse", () => {
  const root = repositoryFixture();
  try {
    const persistent = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      tool_input: { command: "zsh -f", tty: true },
      tool_name: "Bash",
    });
    assert.equal(persistent.status, 0, persistent.stderr);
    assert.match(
      JSON.parse(persistent.stdout).hookSpecificOutput.permissionDecisionReason,
      /Persistent shell sessions/,
    );

    const finite = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      tool_input: { command: "zsh -lc 'git status --short'", tty: true },
      tool_name: "Bash",
    });
    assert.equal(finite.status, 0, finite.stderr);
    assert.equal(finite.stdout, "");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("Codex SessionStart injects the detected native workflow", () => {
  const root = repositoryFixture();
  const fake = fakeCommandPath("codex");
  try {
    const session = runHook(root, {
      cwd: root,
      hook_event_name: "SessionStart",
      model: "test-model",
      source: "startup",
    }, "codex", { PATH: fake.path });
    assert.equal(session.status, 0, session.stderr);
    const output = JSON.parse(session.stdout).hookSpecificOutput;
    assert.equal(output.hookEventName, "SessionStart");
    assert.match(output.additionalContext, /surface="codex"/);
    assert.match(output.additionalContext, /codex review --base origin\/main/);
  } finally {
    fake.cleanup();
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("Codex project hook resolves the shared adapter from a nested working directory", () => {
  const config = JSON.parse(fs.readFileSync(path.join(PROJECT_ROOT, ".codex", "hooks.json"), "utf8"));
  const command = config.hooks.SessionStart[0].hooks[0].command;
  const nested = spawnSync(command, {
    cwd: path.join(PROJECT_ROOT, "web", "ggapi"),
    shell: true,
    input: JSON.stringify({
      cwd: path.join(PROJECT_ROOT, "web", "ggapi"),
      hook_event_name: "SessionStart",
      model: "test-model",
      source: "startup",
    }),
    encoding: "utf8",
  });
  assert.equal(nested.status, 0, nested.stderr);
  assert.equal(JSON.parse(nested.stdout).hookSpecificOutput.hookEventName, "SessionStart");
});

test("native hooks mechanically preserve existing main and upstream rules", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: materialize reviewed tree");
    recordReviewGate(root, { reviewer: "codex", state: "clean", surface: "codex" });
    const upstreamPush = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      model: "test-model",
      tool_input: { cmd: "git push upstream HEAD" },
      tool_name: "exec_command",
      turn_id: "turn",
    });
    assert.match(
      JSON.parse(upstreamPush.stdout).hookSpecificOutput.permissionDecisionReason,
      /forbids every push to upstream/,
    );

    git(root, "switch", "main");
    fs.writeFileSync(path.join(root, "tracked.txt"), "main change\n");
    const mainCommit = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      model: "test-model",
      tool_input: { cmd: "git commit -am 'bad main commit'" },
      tool_name: "exec_command",
      turn_id: "turn",
    });
    assert.match(
      JSON.parse(mainCommit.stdout).hookSpecificOutput.permissionDecisionReason,
      /forbids committing directly on main/,
    );

    const implicitMainPush = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      tool_input: { cmd: "git push origin HEAD" },
      tool_name: "exec_command",
      turn_id: "turn",
    });
    assert.match(
      JSON.parse(implicitMainPush.stdout).hookSpecificOutput.permissionDecisionReason,
      /forbids pushing directly from main/,
    );
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("records clean reviews only for a clean materialized tree", () => {
  const root = repositoryFixture();
  try {
    assert.throws(
      () => recordReviewGate(root, { reviewer: "codex", state: "clean", surface: "codex" }),
      /worktree residue/,
    );
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: materialize reviewed tree");
    recordReviewGate(root, { reviewer: "codex", state: "clean", surface: "codex" });

    fs.writeFileSync(path.join(root, "uncommitted.txt"), "not part of the pushed tree\n");
    assert.equal(reviewGateStatus(root).state, "stale");

    const push = classifyShipMutations(["git push -u origin HEAD"])[0];
    const decision = evaluateShipMutation(root, push);
    assert.equal(decision.allowed, true);
    assert.equal(decision.status.state, "clean");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("requires PR merges to pin a reviewed head for CLI and REST paths", () => {
  const root = repositoryFixture();
  try {
    git(root, "add", "tracked.txt");
    git(root, "commit", "-m", "feat: reviewed head");
    recordReviewGate(root, { reviewer: "codex", state: "clean", surface: "codex" });
    const head = git(root, "rev-parse", "HEAD");

    const unpinned = classifyShipMutations(["gh pr merge 123 --repo AkumaRealLabs/ggapi"])[0];
    assert.match(evaluateShipMutation(root, unpinned).reason, /pin the reviewed head/);

    const pinned = classifyShipMutations([
      `gh pr merge 123 --repo AkumaRealLabs/ggapi --match-head-commit ${head}`,
    ])[0];
    assert.equal(evaluateShipMutation(root, pinned).allowed, true);

    const rest = classifyShipMutations([
      `gh api -X PUT repos/AkumaRealLabs/ggapi/pulls/123/merge -f merge_method=merge -f sha=${head}`,
    ])[0];
    assert.equal(rest.kind, "merge-rest");
    assert.equal(evaluateShipMutation(root, rest).allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("allows only the documented temporary materialization commit before review", () => {
  const root = repositoryFixture();
  try {
    const temporary = runHook(root, {
      cwd: root,
      hook_event_name: "PreToolUse",
      model: "test-model",
      tool_input: { cmd: "git commit -m 'chore: temp codex gate snapshot'" },
      tool_name: "exec_command",
      turn_id: "turn",
    });
    assert.equal(temporary.status, 0, temporary.stderr);
    assert.equal(temporary.stdout, "");
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});

test("allows standalone literal release tag commands on the origin main tip", () => {
  const root = repositoryFixture();
  try {
    git(root, "restore", "tracked.txt");
    git(root, "switch", "main");
    const head = git(root, "rev-parse", "HEAD");
    const tagName = "v1.0.0-rc.20.1";
    const tag = classifyShipMutations([
      `git tag -a ${tagName} -m 'release based on upstream v1.0.0-rc.20' ${head}`,
    ])[0];
    assert.equal(evaluateShipMutation(root, tag).allowed, true);

    git(root, "tag", "-a", tagName, "-m", "release", head);
    const push = classifyShipMutations([
      `git push origin refs/tags/${tagName}:refs/tags/${tagName}`,
    ])[0];
    assert.equal(evaluateShipMutation(root, push).allowed, true);
  } finally {
    fs.rmSync(root, { force: true, recursive: true });
  }
});
