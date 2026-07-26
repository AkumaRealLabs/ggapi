#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

import { describeRuntime, detectSurface, formatReviewCommand } from "./lib/runtime.mjs";
import {
  classifyShipMutations,
  evaluateShipMutation,
  extractShellCommands,
  findRepositoryRoot,
  isShellTool,
  recordReviewGate,
  reviewGateStatus,
  shellToolNames,
  startsPersistentShellSession,
} from "./lib/review-gate.mjs";

function parseArguments(argv) {
  const options = { base: "origin/main", cwd: process.cwd(), json: false, surface: "auto" };
  const positional = [];
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--json") {
      options.json = true;
    } else if (["--base", "--cwd", "--reason", "--reviewer", "--surface"].includes(argument)) {
      if (!argv[index + 1]) throw new Error(`${argument} requires a value`);
      options[argument.slice(2)] = argv[index + 1];
      index += 1;
    } else {
      positional.push(argument);
    }
  }
  return { command: positional[0] || "detect", options };
}

function readHookInput() {
  const raw = fs.readFileSync(0, "utf8").trim();
  return raw ? JSON.parse(raw) : {};
}

function writeJson(value) {
  process.stdout.write(`${JSON.stringify(value)}\n`);
}

function hookEvent(input) {
  const value = input.hook_event_name || input.hookEventName || process.env.GROK_HOOK_EVENT || "";
  const normalized = String(value).replaceAll("_", "").toLowerCase();
  if (normalized === "sessionstart") return "SessionStart";
  if (normalized === "pretooluse") return "PreToolUse";
  return String(value);
}

function hookRoot(input, fallback) {
  const sessionCwd = input.cwd || input.workspaceRoot || process.env.CLAUDE_PROJECT_DIR || fallback;
  // A tool call may name its own working directory (Codex shell.workdir). The
  // gate must validate the repository the command actually runs in, otherwise
  // another worktree's unreviewed tip ships against this repo's clean marker.
  const toolInput = input.tool_input ?? input.toolInput;
  const workdir = toolInput && typeof toolInput === "object"
    ? toolInput.workdir || toolInput.cwd || toolInput.directory
    : null;
  if (typeof workdir === "string" && workdir) {
    try {
      return findRepositoryRoot(path.resolve(sessionCwd, workdir));
    } catch {
      // A workdir outside any repository cannot ship anything; fall back to the
      // session repo rather than denying every read-only call that names one.
    }
  }
  return findRepositoryRoot(sessionCwd);
}

function sessionContext(root, runtime, base) {
  const reviewCommand = formatReviewCommand(runtime.review, base) || "no native review command available";
  const reviewer = runtime.review.reviewer || "codex";
  const relativeScript = path.relative(root, fileURLToPath(import.meta.url)).replaceAll(path.sep, "/");
  return [
    `<ggapi-agent-runtime surface="${runtime.surface}">`,
    "Project-native hooks are active. AGENTS.md and docs/fork are the canonical rules.",
    "Load ggapi-fork for commit, push, PR, merge, release, or upstream-sync work.",
    `Native review command: ${reviewCommand}`,
    `After a clean full-tree review: node ${relativeScript} gate-record --surface ${runtime.surface} --reviewer ${reviewer} --base ${base}`,
    "A changed final tree or moved base invalidates the local gate record automatically.",
    "Only an explicit user review opt-out may be recorded with gate-bypass --reason <reason>.",
    "Hooks never grant commit, push, merge, tag, or deployment authority by themselves.",
    "</ggapi-agent-runtime>",
  ].join("\n");
}

function denyHook(surface, reason) {
  if (surface === "grok") {
    writeJson({ decision: "deny", reason });
    return;
  }
  writeJson({
    hookSpecificOutput: {
      hookEventName: "PreToolUse",
      permissionDecision: "deny",
      permissionDecisionReason: reason,
      additionalContext: reason,
    },
  });
}

function runHook(options) {
  let input = {};
  let surface = options.surface;
  try {
    input = readHookInput();
    surface = detectSurface({ explicit: options.surface, hookInput: input });
    const root = hookRoot(input, options.cwd);
    const event = hookEvent(input);

    if (event === "SessionStart") {
      if (surface === "grok") return;
      const runtime = describeRuntime({ explicit: surface, hookInput: input, root });
      writeJson({
        hookSpecificOutput: {
          hookEventName: "SessionStart",
          additionalContext: sessionContext(root, runtime, options.base),
        },
      });
      return;
    }

    const toolName = input.tool_name || input.toolName || "";
    const toolInput = input.tool_input ?? input.toolInput;
    // A payload carrying a tool call is checked even when its event name is
    // missing or spelled differently; returning silently reads as approval.
    if (event !== "PreToolUse" && !toolName && toolInput === undefined) return;
    if (isShellTool(toolName) && toolInput === undefined) {
      denyHook(surface, "Shell tool input was missing, so the ggapi hook cannot verify the full operation");
      return;
    }
    if ((input.tool_input_truncated || input.toolInputTruncated) && isShellTool(toolName)) {
      denyHook(surface, "Shell tool input was truncated, so the ggapi hook cannot verify the full operation");
      return;
    }
    const commands = extractShellCommands(toolName, toolInput);
    if (startsPersistentShellSession(commands)) {
      denyHook(surface, "Persistent shell sessions are blocked because later stdin is not covered by every runtime's PreToolUse hook");
      return;
    }
    const actions = classifyShipMutations(commands);
    for (const action of actions) {
      const decision = evaluateShipMutation(root, action, options.base);
      if (decision.allowed) continue;

      const runtime = describeRuntime({ explicit: surface, hookInput: input, root });
      const reviewCommand = formatReviewCommand(runtime.review, options.base) || "install or repair a native review CLI first";
      const reviewer = runtime.review.reviewer || "codex";
      const relativeScript = path.relative(root, fileURLToPath(import.meta.url)).replaceAll(path.sep, "/");
      const gateHelp = decision.status
        ? [
            `Run the full-tree review: ${reviewCommand}`,
            `After it is clean, record this exact tree: node ${relativeScript} gate-record --surface ${surface} --reviewer ${reviewer} --base ${options.base}`,
            "Use gate-bypass only after the user explicitly says to skip review.",
          ].join(" ")
        : "This project policy cannot be bypassed by the review gate.";
      denyHook(surface, `${decision.reason}. ${gateHelp}`);
      return;
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    denyHook(detectSurface({ explicit: surface, hookInput: input }), `ggapi hook could not verify the operation: ${message}`);
  }
}

function printResult(value, json) {
  if (json) {
    process.stdout.write(`${JSON.stringify(value, null, 2)}\n`);
    return;
  }
  if (typeof value === "string") process.stdout.write(`${value}\n`);
  else process.stdout.write(`${JSON.stringify(value, null, 2)}\n`);
}

function main() {
  const { command, options } = parseArguments(process.argv.slice(2));
  if (command === "hook") {
    runHook(options);
    return;
  }

  const root = findRepositoryRoot(options.cwd);
  const runtime = describeRuntime({ explicit: options.surface, root });
  if (command === "detect") {
    printResult({
      ...runtime,
      reviewCommand: formatReviewCommand(runtime.review, options.base),
      // Enforcement only covers tool calls with these names.
      enforcedShellTools: shellToolNames(),
    }, options.json);
  } else if (command === "review-command") {
    const reviewCommand = formatReviewCommand(runtime.review, options.base);
    if (!reviewCommand) {
      if (runtime.review.kind.endsWith("-in-session")) {
        printResult(`review the diff vs ${options.base} with this agent's own review capability`, options.json);
        return;
      }
      throw new Error("No native review implementation is available in this environment");
    }
    printResult(reviewCommand, options.json);
  } else if (command === "gate-status") {
    printResult(reviewGateStatus(root, options.base), options.json);
  } else if (command === "gate-record") {
    printResult(recordReviewGate(root, {
      base: options.base,
      reviewer: options.reviewer,
      state: "clean",
      surface: runtime.surface,
    }), options.json);
  } else if (command === "gate-bypass") {
    printResult(recordReviewGate(root, {
      base: options.base,
      reason: options.reason,
      state: "bypassed",
      surface: runtime.surface,
    }), options.json);
  } else {
    throw new Error(`Unknown command: ${command}`);
  }
}

try {
  main();
} catch (error) {
  const message = error instanceof Error ? error.message : String(error);
  process.stderr.write(`${message}\n`);
  process.exitCode = 1;
}
