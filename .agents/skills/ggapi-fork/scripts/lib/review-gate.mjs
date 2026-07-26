import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";

const SHELL_TOOLS = new Set([
  "Bash",
  "Shell",
  "exec",
  "exec_command",
  "functions.exec",
  "run_terminal_command",
  "shell",
  "shell_command",
  // Harnesses that do route stdin through PreToolUse must have it inspected;
  // the persistent-shell block covers the ones that do not.
  "write_stdin",
]);
const JAVASCRIPT_SHELL_TOOLS = new Set(["exec", "functions.exec"]);
const NESTED_SHELLS = new Set(["bash", "dash", "fish", "ksh", "pwsh", "sh", "zsh"]);
// Interpreters that run a program straight from an argument. Their payload is
// not shell, so it cannot be tokenized — a ship mutation inside one is treated
// as uninspectable rather than ignored.
const INLINE_INTERPRETERS = new Set(["node", "deno", "bun", "python", "python3", "perl", "ruby", "php"]);
const MENTIONS_SHIP_MUTATION = /\b(?:git|gh)\b/;
// gh api paths that publish or move release surface, plus graphql, whose query
// text can carry mergePullRequest and friends.
const GH_API_RELEASE_SURFACE = /(?:^|\/)(?:git\/refs(?:\/|$)|releases(?:\/|$))|^graphql$/;
const TEMP_REVIEW_MESSAGE = "chore: temp codex gate snapshot";
const UNRESOLVED_SHIP_COMMAND = "__ggapi_unresolved_ship_command__";
// The gate's own marker lives in the worktree, so every tree/residue check must
// exclude it explicitly instead of relying on .gitignore, which is absent on
// branches and worktrees that predate it.
const MARKER_DIRECTORY = ".ggapi-agent";

function runGit(root, args, extraEnv = {}) {
  const result = spawnSync("git", args, {
    cwd: root,
    env: { ...process.env, ...extraEnv },
    encoding: "utf8",
  });
  if (result.status !== 0) {
    const detail = String(result.stderr || result.stdout || "").trim();
    throw new Error(`git ${args.join(" ")} failed${detail ? `: ${detail}` : ""}`);
  }
  return result.stdout.trim();
}

export function findRepositoryRoot(cwd = process.cwd()) {
  return runGit(cwd, ["rev-parse", "--show-toplevel"]);
}

function currentBranch(root) {
  const result = spawnSync("git", ["symbolic-ref", "--quiet", "--short", "HEAD"], {
    cwd: root,
    encoding: "utf8",
  });
  return result.status === 0 ? result.stdout.trim() : `detached:${runGit(root, ["rev-parse", "--short", "HEAD"])}`;
}

function finalTree(root) {
  const temporaryRoot = fs.mkdtempSync(path.join(os.tmpdir(), "ggapi-review-tree-"));
  const temporaryObjects = path.join(temporaryRoot, "objects");
  fs.mkdirSync(temporaryObjects);

  let repositoryObjects = runGit(root, ["rev-parse", "--git-path", "objects"]);
  if (!path.isAbsolute(repositoryObjects)) repositoryObjects = path.resolve(root, repositoryObjects);

  const indexEnvironment = {
    GIT_INDEX_FILE: path.join(temporaryRoot, "index"),
    GIT_OBJECT_DIRECTORY: temporaryObjects,
    GIT_ALTERNATE_OBJECT_DIRECTORIES: repositoryObjects,
  };

  try {
    runGit(root, ["read-tree", "HEAD"], indexEnvironment);
    runGit(root, ["add", "-A", "--", "."], indexEnvironment);
    // Drop the gate's own marker even on branches whose .gitignore lacks it,
    // where "add -A" would otherwise fold it into the reviewed tree.
    runGit(root, ["rm", "--cached", "-r", "--quiet", "--ignore-unmatch", "--", MARKER_DIRECTORY], indexEnvironment);
    return runGit(root, ["write-tree"], indexEnvironment);
  } finally {
    fs.rmSync(temporaryRoot, { force: true, recursive: true });
  }
}

function stagedTree(root) {
  // "git commit" writes the index, which can hold content the worktree no
  // longer has. Hash a copy of the real index so the commit path compares the
  // tree that will actually land, not the worktree snapshot.
  const temporaryRoot = fs.mkdtempSync(path.join(os.tmpdir(), "ggapi-review-index-"));
  const temporaryObjects = path.join(temporaryRoot, "objects");
  fs.mkdirSync(temporaryObjects);

  let repositoryObjects = runGit(root, ["rev-parse", "--git-path", "objects"]);
  if (!path.isAbsolute(repositoryObjects)) repositoryObjects = path.resolve(root, repositoryObjects);
  let repositoryIndex = runGit(root, ["rev-parse", "--git-path", "index"]);
  if (!path.isAbsolute(repositoryIndex)) repositoryIndex = path.resolve(root, repositoryIndex);

  const indexCopy = path.join(temporaryRoot, "index");
  const indexEnvironment = {
    GIT_INDEX_FILE: indexCopy,
    GIT_OBJECT_DIRECTORY: temporaryObjects,
    GIT_ALTERNATE_OBJECT_DIRECTORIES: repositoryObjects,
  };

  try {
    if (fs.existsSync(repositoryIndex)) fs.copyFileSync(repositoryIndex, indexCopy);
    else runGit(root, ["read-tree", "HEAD"], indexEnvironment);
    runGit(root, ["rm", "--cached", "-r", "--quiet", "--ignore-unmatch", "--", MARKER_DIRECTORY], indexEnvironment);
    return runGit(root, ["write-tree"], indexEnvironment);
  } finally {
    fs.rmSync(temporaryRoot, { force: true, recursive: true });
  }
}

function fingerprintFor(base, baseOid, branch, tree) {
  return crypto
    .createHash("sha256")
    .update(JSON.stringify({ base, baseOid, branch, tree }))
    .digest("hex");
}

function markerPath(root) {
  return path.join(root, MARKER_DIRECTORY, "review-gate.json");
}

export function calculateReviewFingerprint(root, base = "origin/main") {
  const baseOid = runGit(root, ["rev-parse", "--verify", `${base}^{commit}`]);
  const baseTree = runGit(root, ["rev-parse", "--verify", `${base}^{tree}`]);
  const tree = finalTree(root);
  const branch = currentBranch(root);
  const fingerprint = fingerprintFor(base, baseOid, branch, tree);

  return {
    base,
    baseOid,
    baseTree,
    branch,
    fingerprint,
    reviewable: tree !== baseTree,
    tree,
  };
}

function readMarker(root) {
  try {
    return JSON.parse(fs.readFileSync(markerPath(root), "utf8"));
  } catch {
    return null;
  }
}

function reviewGateStatusForTree(root, tree, base = "origin/main", branch = currentBranch(root)) {
  const baseOid = runGit(root, ["rev-parse", "--verify", `${base}^{commit}`]);
  const baseTree = runGit(root, ["rev-parse", "--verify", `${base}^{tree}`]);
  const current = {
    base,
    baseOid,
    baseTree,
    branch,
    fingerprint: fingerprintFor(base, baseOid, branch, tree),
    reviewable: tree !== baseTree,
    tree,
  };
  if (!current.reviewable) return { ...current, valid: true, state: "empty", marker: null };

  const marker = readMarker(root);
  if (!marker) return { ...current, valid: false, state: "missing", marker: null };
  if (marker.fingerprint !== current.fingerprint) {
    return { ...current, valid: false, state: "stale", marker };
  }
  if (marker.state !== "clean" && marker.state !== "bypassed") {
    return { ...current, valid: false, state: "invalid", marker };
  }
  return { ...current, valid: true, state: marker.state, marker };
}

export function reviewGateStatus(root, base = "origin/main") {
  return reviewGateStatusForTree(root, finalTree(root), base);
}

export function recordReviewGate(root, {
  base = "origin/main",
  reason = null,
  reviewer,
  state = "clean",
  surface = "generic",
} = {}) {
  if (state !== "clean" && state !== "bypassed") throw new Error(`Unsupported gate state: ${state}`);
  if (state === "clean" && !reviewer) throw new Error("A reviewer is required for a clean gate record");
  if (state === "bypassed" && !String(reason || "").trim()) {
    throw new Error("An explicit user-approved reason is required to bypass the review gate");
  }
  if (state === "clean") {
    const residue = runGit(root, ["status", "--porcelain=v1", "--untracked-files=normal"])
      .split("\n")
      .filter((line) => line.trim() && !line.slice(3).replace(/^"/, "").startsWith(`${MARKER_DIRECTORY}/`));
    if (residue.length) {
      throw new Error("Cannot record a clean review while tracked or untracked worktree residue remains");
    }
  }

  const current = calculateReviewFingerprint(root, base);
  const marker = {
    schemaVersion: 1,
    state,
    surface,
    reviewer: reviewer || null,
    reason: reason ? String(reason).trim() : null,
    recordedAt: new Date().toISOString(),
    base: current.base,
    baseOid: current.baseOid,
    branch: current.branch,
    tree: current.tree,
    fingerprint: current.fingerprint,
  };

  const destination = markerPath(root);
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  const temporary = `${destination}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(marker, null, 2)}\n`, { mode: 0o600 });
  fs.renameSync(temporary, destination);
  return marker;
}

function javascriptStringEnd(source, start) {
  const quote = source[start];
  let escaped = false;
  for (let index = start + 1; index < source.length; index += 1) {
    const character = source[index];
    if (escaped) {
      escaped = false;
      continue;
    }
    if (character === "\\") {
      escaped = true;
      continue;
    }
    if (character === quote) return index;
  }
  return -1;
}

function javascriptToolCalls(source) {
  const calls = [];
  const pattern = /\btools\.(exec_command|write_stdin)\s*\(/g;
  // Driven with exec (not matchAll) so advancing lastIndex past a consumed call
  // actually skips it; matchAll iterates a clone and would rescan nested calls.
  for (let match = pattern.exec(source); match; match = pattern.exec(source)) {
    const start = match.index + match[0].length;
    let depth = 1;
    let lineComment = false;
    let blockComment = false;
    for (let index = start; index < source.length; index += 1) {
      const character = source[index];
      const next = source[index + 1];
      if (lineComment) {
        if (character === "\n") lineComment = false;
        continue;
      }
      if (blockComment) {
        if (character === "*" && next === "/") {
          blockComment = false;
          index += 1;
        }
        continue;
      }
      if (character === "/" && next === "/") {
        lineComment = true;
        index += 1;
        continue;
      }
      if (character === "/" && next === "*") {
        blockComment = true;
        index += 1;
        continue;
      }
      if (["'", '"', "`"].includes(character)) {
        const end = javascriptStringEnd(source, index);
        if (end < 0) break;
        index = end;
        continue;
      }
      if (character === "(") depth += 1;
      else if (character === ")") {
        depth -= 1;
        if (depth === 0) {
          calls.push({ body: source.slice(start, index), method: match[1] });
          pattern.lastIndex = index + 1;
          break;
        }
      }
    }
  }
  return calls;
}

function javascriptProperty(body, names, constants) {
  for (const name of names) {
    const occurrences = body.match(new RegExp(`\\b${name}\\s*:`, "g"));
    if (!occurrences) continue;
    // JavaScript keeps the LAST duplicate key, so a payload carrying several
    // is not something this reader may resolve to the first one.
    if (occurrences.length > 1) return { present: true, resolved: false };
    const property = new RegExp(`\\b${name}\\s*:`).exec(body);
    let index = property.index + property[0].length;
    while (/\s/.test(body[index] || "")) index += 1;
    if (["'", '"', "`"].includes(body[index])) {
      const end = javascriptStringEnd(body, index);
      if (end < 0) return { present: true, resolved: false };
      const value = decodeJavascriptString(body.slice(index, end + 1));
      if (value === null) return { present: true, resolved: false };
      // A literal that is only the first operand of an expression ("git" + rest)
      // does not describe the command that will actually run.
      let after = end + 1;
      while (/\s/.test(body[after] || "")) after += 1;
      if (body[after] && ![",", "}"].includes(body[after])) return { present: true, resolved: false };
      return { present: true, resolved: true, value };
    }
    const identifier = /^[A-Za-z_$][\w$]*/.exec(body.slice(index))?.[0];
    if (identifier && constants.has(identifier)) {
      // Same guard as the literal branch: a constant that is only the first
      // operand of an expression does not describe the command that will run.
      let after = index + identifier.length;
      while (/\s/.test(body[after] || "")) after += 1;
      if (body[after] && ![",", "}"].includes(body[after])) return { present: true, resolved: false };
      return { present: true, resolved: true, value: constants.get(identifier) };
    }
    return { present: true, resolved: false };
  }

  for (const name of names) {
    if (new RegExp(`\\b${name}\\s*(?=[,}])`).test(body)) {
      return constants.has(name)
        ? { present: true, resolved: true, value: constants.get(name) }
        : { present: true, resolved: false };
    }
  }
  return { present: false, resolved: false };
}

function javascriptCommandStrings(source) {
  const commands = [];
  const constants = new Map();
  const constantPattern = /\bconst\s+([A-Za-z_$][\w$]*)\s*=\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`(?:\\.|[^`\\])*`)/gs;
  for (const match of source.matchAll(constantPattern)) {
    const value = decodeJavascriptString(match[2]);
    if (value !== null) constants.set(match[1], value);
  }

  const calls = javascriptToolCalls(source);
  const allToolCalls = [...source.matchAll(/\btools\.[A-Za-z_$][\w$]*\s*\(/g)].length;
  let unresolved = false;
  for (const call of calls) {
    if (!call.body.trim().startsWith("{")) {
      unresolved = true;
      continue;
    }
    const property = call.method === "exec_command"
      ? javascriptProperty(call.body, ["cmd", "command"], constants)
      : javascriptProperty(call.body, ["chars"], constants);
    if (!property.present && call.method === "write_stdin") continue;
    if (!property.resolved) {
      unresolved = true;
      continue;
    }
    commands.push(property.value);
  }
  // Kept in step with the shell-side heuristic in classifyShipMutations.
  const mentionsShipMutation = (/\bgit\b/.test(source) && /\b(?:commit|push|tag)\b/.test(source))
    || (/\bgh\b/.test(source) && /\bpr\b/.test(source) && /\bmerge\b/.test(source))
    || /\/pulls\/[^\s"'`]+\/merge\b/.test(source)
    || (/\bgh\b/.test(source) && (/\brelease\b/.test(source) || /git\/refs/.test(source)));
  const containsShipMutation = commands.some((command) => classifyShipMutations([command]).length > 0);
  if (unresolved
    || (mentionsShipMutation && calls.length === 0)
    || (containsShipMutation && (calls.length > 1 || allToolCalls > calls.length))) {
    commands.push(UNRESOLVED_SHIP_COMMAND);
  }
  return commands;
}

function decodeJavascriptString(literal) {
  try {
    if (literal.startsWith("`") && /(^|[^\\])(?:\\\\)*\$\{/.test(literal.slice(1, -1))) return null;
    if (literal.startsWith('"')) return JSON.parse(literal);
    return literal
      .slice(1, -1)
      .replace(/\\r\\n|\\n|\\r/g, "\n")
      .replace(/\\t/g, "\t")
      .replace(/\\(['"`\\])/g, "$1");
  } catch {
    return null;
  }
}

export function isShellTool(toolName) {
  return SHELL_TOOLS.has(toolName);
}

// The gate only sees tool calls whose name it recognizes. A harness that names
// its exec tool something else gets zero enforcement while the hook still looks
// installed, so the roster is reported by `detect` rather than left implicit.
export function shellToolNames() {
  return [...SHELL_TOOLS].sort();
}

function shellQuoteArgument(argument) {
  if (/^[A-Za-z0-9_./:@=+,^-]*$/.test(argument)) return argument;
  return `'${argument.replaceAll("'", `'"'"'`)}'`;
}

export function extractShellCommands(toolName, toolInput) {
  if (!isShellTool(toolName)) return [];
  if (typeof toolInput === "string") {
    return JAVASCRIPT_SHELL_TOOLS.has(toolName) ? javascriptCommandStrings(toolInput) : [toolInput];
  }
  if (!toolInput || typeof toolInput !== "object") return [];

  const commands = [];
  for (const key of ["cmd", "command", "script", "chars"]) {
    const value = toolInput[key];
    if (value === undefined || value === null) continue;
    if (typeof value === "string") {
      commands.push(value);
    } else if (Array.isArray(value) && value.every((entry) => typeof entry === "string")) {
      // argv form (Codex shell tool). Quote each element so re-tokenizing yields
      // the original argv; a plain join would split "-lc <script>" into loose
      // words and hide the nested command.
      commands.push(value.map(shellQuoteArgument).join(" "));
    } else {
      // A shell payload we cannot read must never fail open.
      commands.push(UNRESOLVED_SHIP_COMMAND);
    }
  }
  if (JAVASCRIPT_SHELL_TOOLS.has(toolName)) {
    for (const key of ["code", "input"]) {
      if (typeof toolInput[key] === "string") commands.push(...javascriptCommandStrings(toolInput[key]));
    }
  }
  return commands;
}

export function startsPersistentShellSession(commands) {
  const valueOptions = new Set(["-O", "-o", "--init-file", "--rcfile"]);
  for (const command of commands) {
    // Every segment matters: "true; bash" also opens an interactive shell.
    for (const segment of splitShellSegments(command)) {
      const tokens = commandTokens(segment);
      if (!NESTED_SHELLS.has(path.basename(tokens[0] || ""))) continue;

      let hasScript = false;
      for (let index = 1; index < tokens.length; index += 1) {
        const token = tokens[index];
        // These print and exit, so they never open an interactive session.
        if (["--version", "-V", "--help"].includes(token)) {
          hasScript = true;
          break;
        }
        if (/^-[^-]*c/.test(token) || ["--command", "-Command", "-File"].includes(token)) {
          hasScript = true;
          break;
        }
        if (valueOptions.has(token)) {
          index += 1;
          continue;
        }
        if (!token.startsWith("-")) {
          hasScript = true;
          break;
        }
      }
      if (!hasScript) return true;
    }
  }
  return false;
}

function shellTokens(segment) {
  // Adjacent quoted and bare fragments form ONE shell word: git ""commit and
  // git c"o"mmit both run "commit". Emitting a token per fragment would let
  // that spelling walk past every subcommand comparison.
  const tokens = [];
  let current = null;
  let index = 0;
  while (index < segment.length) {
    const character = segment[index];
    if (/\s/.test(character)) {
      if (current !== null) tokens.push(current);
      current = null;
      index += 1;
      continue;
    }
    if (current === null) current = "";
    if (character === "\\") {
      if (index + 1 < segment.length) current += segment[index + 1];
      index += 2;
      continue;
    }
    if (character === "'") {
      const end = segment.indexOf("'", index + 1);
      current += end < 0 ? segment.slice(index + 1) : segment.slice(index + 1, end);
      index = end < 0 ? segment.length : end + 1;
      continue;
    }
    if (character === '"') {
      let scan = index + 1;
      while (scan < segment.length && segment[scan] !== '"') {
        if (segment[scan] === "\\" && scan + 1 < segment.length) {
          current += segment[scan + 1];
          scan += 2;
          continue;
        }
        current += segment[scan];
        scan += 1;
      }
      index = scan + 1;
      continue;
    }
    current += character;
    index += 1;
  }
  if (current !== null) tokens.push(current);
  return tokens;
}

// Environment that redirects git at another repository, index, or config is
// equivalent to the --git-dir/-c options gitInvocationViolation already rejects.
// GIT_CONFIG_COUNT/KEY_n/VALUE_n and GIT_CONFIG_PARAMETERS reach exactly as far
// as the explicitly blocked `git -c` (e.g. rewriting remote.origin.url).
const GIT_REDIRECTING_ENVIRONMENT = /^GIT_(?:DIR|WORK_TREE|INDEX_FILE|OBJECT_DIRECTORY|ALTERNATE_OBJECT_DIRECTORIES|COMMON_DIR|CEILING_DIRECTORIES|CONFIG(?:_[A-Z0-9_]+)?)=/;

function leadingAssignments(segment) {
  const assignments = [];
  for (const token of shellTokens(segment)) {
    if (/^[A-Za-z_][A-Za-z0-9_]*=/.test(token)) {
      assignments.push(token);
      continue;
    }
    if (["command", "env", "sudo"].includes(token)) continue;
    break;
  }
  return assignments;
}

function commandTokens(segment) {
  let tokens = shellTokens(segment);
  while (tokens[0] && (/^[A-Za-z_][A-Za-z0-9_]*=.*/.test(tokens[0]) || ["command", "env", "sudo"].includes(tokens[0]))) {
    tokens = tokens.slice(1);
  }

  return tokens;
}

function splitShellSegments(command) {
  const segments = [];
  let current = "";
  let quote = null;
  let escaped = false;
  for (let index = 0; index < command.length; index += 1) {
    const character = command[index];
    if (escaped) {
      current += character;
      escaped = false;
      continue;
    }
    if (character === "\\" && quote !== "'") {
      current += character;
      escaped = true;
      continue;
    }
    if (quote) {
      current += character;
      if (character === quote) quote = null;
      continue;
    }
    if (["'", '"', "`"].includes(character)) {
      quote = character;
      current += character;
      continue;
    }

    const pair = command.slice(index, index + 2);
    if (["&&", "||"].includes(pair)) {
      if (current.trim()) segments.push(current.trim());
      current = "";
      index += 1;
      continue;
    }
    if ([";", "\n", "|"].includes(character)) {
      if (current.trim()) segments.push(current.trim());
      current = "";
      continue;
    }
    // A lone "&" backgrounds the command and starts another one, so it splits
    // segments too. Redirections that merge descriptors ("&>", "2>&1") keep
    // their "&" as part of the same command.
    if (character === "&" && command[index + 1] !== ">" && !/>\s*$/.test(current)) {
      if (current.trim()) segments.push(current.trim());
      current = "";
      continue;
    }
    current += character;
  }
  if (current.trim()) segments.push(current.trim());
  return segments;
}

function normalizeGitTokens(tokens) {
  if (path.basename(tokens[0] || "") !== "git") return { gitOptions: {}, tokens };
  const valueOptions = new Set(["-C", "-c", "--git-dir", "--work-tree", "--namespace", "--super-prefix", "--config-env", "--exec-path"]);
  const gitOptions = { directories: [], overridesRepository: false, overridesConfig: false };
  let index = 1;
  while (index < tokens.length && tokens[index].startsWith("-")) {
    const option = tokens[index];
    if (option === "-C") gitOptions.directories.push(tokens[index + 1] || "");
    else if (option.startsWith("-C") && option.length > 2) gitOptions.directories.push(option.slice(2));
    else if (option === "-c" || option.startsWith("-c") || option === "--config-env" || option.startsWith("--config-env=")) {
      gitOptions.overridesConfig = true;
    } else if (["--git-dir", "--work-tree", "--namespace", "--super-prefix"].some((name) => option === name || option.startsWith(`${name}=`))) {
      gitOptions.overridesRepository = true;
    }
    if (valueOptions.has(option)) {
      index += 2;
    } else if ([...valueOptions].some((candidate) => option.startsWith(`${candidate}=`))) {
      index += 1;
    } else {
      index += 1;
    }
  }
  return { gitOptions, tokens: [tokens[0], ...tokens.slice(index)] };
}

function optionValue(tokens, names) {
  for (let index = 0; index < tokens.length; index += 1) {
    const token = tokens[index];
    if (names.includes(token)) return tokens[index + 1] || null;
    for (const name of names) {
      if (token.startsWith(`${name}=`)) return token.slice(name.length + 1);
      // Short options may be glued to their value: -XPUT is -X PUT.
      if (/^-[A-Za-z]$/.test(name) && token.startsWith(name) && token.length > name.length) {
        return token.slice(name.length);
      }
    }
  }
  return null;
}

// gh api defaults to GET, but switches to POST as soon as a field is supplied,
// so the explicit -X/--method is not the only thing that decides the verb.
function ghApiMethod(tokens) {
  const explicit = optionValue(tokens.slice(2), ["-X", "--method"]);
  if (explicit) return explicit.toUpperCase();
  const sendsFields = tokens.slice(2).some((token) => ["-f", "-F", "--field", "--raw-field", "--input"].includes(token));
  return sendsFields ? "POST" : "GET";
}

// Flags only count when they are the option itself, never when they happen to
// be the VALUE of a preceding option: `git commit -m --dry-run` really commits.
function hasFlag(tokens, flags, valueOptions = ["-m", "--message", "-F", "--file", "-X", "--method", "-C", "--reuse-message"]) {
  for (let index = 0; index < tokens.length; index += 1) {
    const token = tokens[index];
    if (valueOptions.includes(token)) {
      index += 1;
      continue;
    }
    if (flags.includes(token)) return true;
  }
  return false;
}

function exactTemporaryCommit(tokens) {
  const message = optionValue(tokens.slice(2), ["-m", "--message"]);
  if (message !== TEMP_REVIEW_MESSAGE) return false;
  const messageOptions = tokens.slice(2).filter((token) => token === "-m" || token === "--message" || token.startsWith("--message="));
  return messageOptions.length === 1 && !tokens.slice(2).some((token) => token === "-F" || token === "--file" || token.startsWith("--file="));
}

function tagMutates(tokens) {
  const args = tokens.slice(2);
  if (args.length === 0) return false;
  if (hasFlag(args, ["--help", "-h"])) return false;
  if (args.some((arg) => [
    "-a", "--annotate", "-s", "--sign", "-d", "--delete", "-f", "--force",
    "-m", "--message", "-F", "--file",
  ].includes(arg))) {
    return true;
  }
  const listFlags = [
    "-l", "--list", "-n", "--contains", "--format", "--merged", "--no-merged",
    "--points-at", "--sort", "--verify", "-v", "--column", "--ignore-case",
  ];
  if (args.some((arg) => listFlags.some((flag) => arg === flag || arg.startsWith(`${flag}=`)))) {
    return false;
  }
  return args.some((arg) => !arg.startsWith("-"));
}

function tokensDescribeShipMutation(rawTokens) {
  const normalized = normalizeGitTokens(rawTokens);
  const tokens = normalized.tokens;
  const executable = path.basename(tokens[0] || "");
  if (executable === "git" && tokens[1] === "commit") {
    return !hasFlag(tokens, ["--dry-run", "--help", "-h"]);
  }
  if (executable === "git" && tokens[1] === "push") {
    return !hasFlag(tokens, ["--dry-run", "-n", "--help"]);
  }
  if (executable === "git" && tokens[1] === "tag") return tagMutates(tokens);
  if (executable === "gh" && tokens[1] === "pr" && tokens[2] === "merge") {
    return !hasFlag(tokens, ["--help", "-h"]);
  }
  if (executable === "gh" && tokens[1] === "api" && tokens.some((token) => /(?:^|\/)pulls\/[^/]+\/merge$/.test(token))) {
    return ghApiMethod(tokens) === "PUT";
  }
  // Kept in step with classifyShipMutations: this is the only detector for
  // mutations hidden inside grouping or command substitution.
  if (executable === "gh" && tokens[1] === "release" && ["create", "delete", "edit"].includes(tokens[2])) return true;
  if (executable === "gh" && tokens[1] === "api" && tokens.some((token) => /(?:^|\/)git\/refs(?:\/|$)/.test(token))) {
    return ["DELETE", "PATCH", "POST"].includes(ghApiMethod(tokens));
  }
  return false;
}

function nestedTokenWindow(rawTokens, start) {
  let first = rawTokens[start];
  const substitution = first.lastIndexOf("$(");
  if (substitution >= 0) first = first.slice(substitution + 2);
  else first = first.replace(/^[!({`]+/, "");
  let tokens = [first, ...rawTokens.slice(start + 1)].map((token) => token.replace(/[)};`]+$/, ""));
  const outerExecutable = path.basename(rawTokens[0] || "");
  if (tokens[0].includes(" ") && (substitution >= 0 || ["eval", "exec"].includes(outerExecutable))) {
    tokens = [...shellTokens(tokens[0]), ...tokens.slice(1)];
  }
  return tokens;
}

// Bodies of $( ) and ` ` run as full commands of their own. Quoting keeps them
// inside a single segment, so they must be parsed rather than word-scanned.
function commandSubstitutions(segment) {
  const bodies = [];
  for (let index = 0; index < segment.length; index += 1) {
    if (segment[index] === "$" && segment[index + 1] === "(") {
      let depth = 1;
      let scan = index + 2;
      const start = scan;
      while (scan < segment.length && depth > 0) {
        if (segment[scan] === "(") depth += 1;
        else if (segment[scan] === ")") depth -= 1;
        scan += 1;
      }
      if (depth === 0) bodies.push(segment.slice(start, scan - 1));
      index = scan - 1;
      continue;
    }
    if (segment[index] === "`") {
      const end = segment.indexOf("`", index + 1);
      if (end < 0) break;
      bodies.push(segment.slice(index + 1, end));
      index = end;
    }
  }
  return bodies;
}

function containsNestedShipMutation(rawTokens) {
  for (let index = 0; index < rawTokens.length; index += 1) {
    const token = rawTokens[index];
    const substitution = token.includes("$(") || token.startsWith("`") || /^[!({]/.test(token);
    const candidate = nestedTokenWindow(rawTokens, index)[0];
    if (index > 0 || substitution || path.basename(candidate || "") !== path.basename(rawTokens[0] || "")) {
      const nested = nestedTokenWindow(rawTokens, index);
      if (["git", "gh"].includes(path.basename(nested[0] || "")) && tokensDescribeShipMutation(nested)) return true;
    }
  }
  return false;
}

export function classifyShipMutations(commands) {
  const actions = [];
  for (const command of commands) {
    if (command === UNRESOLVED_SHIP_COMMAND) {
      actions.push({ kind: "unknown", command, compound: true, tokens: [] });
      continue;
    }
    const commandActionCount = actions.length;
    const segments = splitShellSegments(command);
    for (const segment of segments) {
      const rawTokens = commandTokens(segment);
      const actionCount = actions.length;
      const outerExecutable = path.basename(rawTokens[0] || "");
      // Redirecting environment applies to the whole invocation, including a
      // nested shell, so it has to be checked before the recursion below.
      const redirectsGit = leadingAssignments(segment).some((assignment) => GIT_REDIRECTING_ENVIRONMENT.test(assignment));
      if (NESTED_SHELLS.has(outerExecutable)) {
        // Any short-option bundle ending in "c" carries the script: -c, -lc, -ec.
        const commandIndex = rawTokens.findIndex((token) => /^-[A-Za-z]*c$/.test(token) || ["--command", "-Command"].includes(token));
        const script = commandIndex >= 0 ? rawTokens[commandIndex + 1] : null;
        if (script) {
          const nested = classifyShipMutations([script]);
          for (const action of nested) {
            actions.push({
              ...action,
              compound: action.compound || segments.length > 1,
              gitOptions: redirectsGit ? { ...action.gitOptions, overridesRepository: true } : action.gitOptions,
            });
          }
        } else if (rawTokens.slice(1).some((token) => MENTIONS_SHIP_MUTATION.test(token))) {
          // A shell payload we cannot read must never fail open.
          actions.push({ kind: "unknown", command: segment, compound: true, tokens: [] });
        }
        continue;
      }
      const normalized = normalizeGitTokens(rawTokens);
      const tokens = normalized.tokens;
      // commandTokens drops the leading VAR=value prefix, so redirecting
      // environment would otherwise never reach gitInvocationViolation.
      const gitOptions = leadingAssignments(segment).some((assignment) => GIT_REDIRECTING_ENVIRONMENT.test(assignment))
        ? { ...normalized.gitOptions, overridesRepository: true }
        : normalized.gitOptions;
      const executable = path.basename(tokens[0] || "");
      if (executable === "git" && tokens[1] === "commit") {
        if (!hasFlag(tokens, ["--dry-run", "--help", "-h"])) {
          actions.push({ kind: "commit", command: segment, compound: segments.length > 1, gitOptions, temporarySnapshot: exactTemporaryCommit(tokens), tokens });
        }
      } else if (executable === "git" && tokens[1] === "push") {
        if (!hasFlag(tokens, ["--dry-run", "-n", "--help"])) {
          actions.push({ kind: "push", command: segment, compound: segments.length > 1, gitOptions, tokens });
        }
      } else if (executable === "git" && tokens[1] === "tag" && tagMutates(tokens)) {
        actions.push({ kind: "tag", command: segment, compound: segments.length > 1, gitOptions, tokens });
      } else if (executable === "gh" && tokens[1] === "pr" && tokens[2] === "merge") {
        if (!hasFlag(tokens, ["--help", "-h"])) {
          actions.push({ kind: "merge", command: segment, compound: segments.length > 1, tokens });
        }
      } else if (executable === "gh" && tokens[1] === "api" && tokens.some((token) => /(?:^|\/)pulls\/[^/]+\/merge$/.test(token))) {
        const method = ghApiMethod(tokens);
        if (method === "PUT") {
          actions.push({ kind: "merge-rest", command: segment, compound: segments.length > 1, tokens });
        }
      } else if (executable === "gh" && tokens[1] === "release" && ["create", "delete", "edit"].includes(tokens[2])) {
        // Creates or moves a remote tag/release outside the documented
        // tag → GHCR path, so it goes through the same tag rule.
        actions.push({ kind: "tag", command: segment, compound: segments.length > 1, gitOptions, tokens });
      } else if (executable === "gh" && tokens[1] === "api" && tokens.some((token) => GH_API_RELEASE_SURFACE.test(token))) {
        // git/refs and releases both publish tags; graphql can carry any mutation.
        const method = ghApiMethod(tokens);
        const graphql = tokens.includes("graphql");
        if (graphql || ["DELETE", "PATCH", "POST"].includes(method)) {
          actions.push({ kind: graphql ? "unknown" : "tag", command: segment, compound: graphql || segments.length > 1, gitOptions, tokens });
        }
      }
      if (INLINE_INTERPRETERS.has(outerExecutable)
        && rawTokens.some((token) => /^-(?:e|c|-eval|-execute)$/.test(token))
        && rawTokens.slice(1).some((token) => MENTIONS_SHIP_MUTATION.test(token))) {
        actions.push({ kind: "unknown", command: segment, compound: true, tokens: [] });
        continue;
      }

      // Always look for a second mutation hidden in the same segment. Checking
      // this only when the segment produced no action would let a recognized
      // (even auto-allowed) command smuggle another one alongside it.
      const substituted = commandSubstitutions(segment)
        .some((body) => classifyShipMutations([body]).length > 0);
      if (substituted || containsNestedShipMutation(rawTokens)) {
        if (actions.length === actionCount) {
          actions.push({ kind: "unknown", command: segment, compound: true, tokens: [] });
        } else {
          for (let index = actionCount; index < actions.length; index += 1) actions[index].compound = true;
        }
      }
    }
    const mentionsShipMutation = (/\bgit\b/.test(command) && /\b(?:commit|push|tag)\b/.test(command))
      || (/\bgh\b/.test(command) && /\bpr\b/.test(command) && /\bmerge\b/.test(command))
      || (/\bgh\b/.test(command) && (/\brelease\b/.test(command) || /pulls\/[^/\s]+\/merge|git\/refs/.test(command)));
    if (actions.length === commandActionCount && mentionsShipMutation && /\$\(|`|\$\{?[A-Za-z_]/.test(command)) {
      actions.push({ kind: "unknown", command, compound: true, tokens: [] });
    }
  }
  return actions;
}

function pushArguments(tokens) {
  const valueOptions = new Set(["--repo", "--receive-pack", "--exec"]);
  const positional = [];
  for (let index = 2; index < tokens.length; index += 1) {
    const token = tokens[index];
    if (valueOptions.has(token)) {
      index += 1;
    } else if (!token.startsWith("-")) {
      positional.push(token);
    }
  }
  return positional;
}

function pushedTree(root, action, branch) {
  const positional = pushArguments(action.tokens);
  const remote = positional[0];
  if (!remote) return { violation: "Pushes must explicitly target origin" };
  if (remote === "upstream") return { violation: "Project policy forbids every push to upstream" };
  if (remote !== "origin") return { violation: "Project policy permits pushes only to origin" };

  const refspecs = positional.slice(1);
  if (refspecs.length === 0) return { violation: "Pushes must name the exact reviewed source ref" };
  if (hasFlag(action.tokens, ["--delete", "-d"])) {
    const deletesMain = refspecs.some(targetsMainBranch);
    if (deletesMain) return { violation: "Project policy forbids deleting main" };
    // Remote tag deletion is a release-surface mutation, like the gh equivalents.
    // A bare "v1.0.0-rc.21.1" deletes the tag just as refs/tags/... does, so ask
    // git whether the name resolves to a tag rather than guessing from its shape.
    // Tag names may contain slashes, so the lookup cannot be skipped by shape.
    if (refspecs.some((ref) => resolvesToTag(root, ref))) {
      return { violation: "Remote tag deletion must go through the release process, not a push --delete" };
    }
    // A published tag deleted locally first (SOP §5 cleanup) no longer resolves
    // here and is indistinguishable from a branch by name alone. Rather than
    // guess, require the caller to say which namespace they mean.
    const ambiguous = refspecs.find((ref) => !ref.startsWith("refs/heads/") && !resolvesToBranch(root, ref));
    if (ambiguous) {
      return { violation: `Spell the ref to delete as refs/heads/${ambiguous} or refs/tags/${ambiguous}; a bare name that is not a known branch is ambiguous` };
    }
    return { deletion: true };
  }

  // Bulk tag pushes publish every matching local tag while only the branch
  // refspec gets checked, so the origin/main tip rule would not apply to them.
  if (action.tokens.includes("--tags") || action.tokens.includes("--follow-tags")) {
    return { violation: "Tag pushes must name the exact tag refspec instead of --tags/--follow-tags" };
  }

  const head = runGit(root, ["rev-parse", "HEAD"]);
  const main = runGit(root, ["rev-parse", "origin/main"]);
  let tree = null;
  for (const refspec of refspecs) {
    const [source, explicitDestination] = refspec.split(":", 2);
    if (!source) return { violation: "Remote ref deletion must use an explicit --delete command" };
    // Without ":dest" the remote ref takes the source name, so "git push origin
    // main" still lands on origin/main.
    const destination = explicitDestination === undefined ? source : explicitDestination;
    if (targetsMainBranch(destination)) {
      return { violation: "Project policy forbids pushing directly to main" };
    }

    const commit = runGit(root, ["rev-parse", "--verify", `${source}^{commit}`]);
    const sourceTree = runGit(root, ["rev-parse", "--verify", `${source}^{tree}`]);
    // A bare "v9.9.9" publishes the tag just as refs/tags/v9.9.9 does, so the
    // tip rule has to resolve names the same way the --delete path does.
    if (source.startsWith("refs/tags/") || destination.startsWith("refs/tags/") || resolvesToTag(root, source)) {
      if (branch !== "main" || head !== main || commit !== main) {
        return { violation: "Release tags must point at the origin/main tip" };
      }
      continue;
    }
    if (branch === "main") return { violation: "Project policy forbids pushing directly from main" };
    if (commit !== head) return { violation: "Push source must be the reviewed current branch tip" };
    if (tree && tree !== sourceTree) return { violation: "A single push cannot ship multiple reviewed trees" };
    tree = sourceTree;
  }
  return { tree };
}

function mergeTree(root, action) {
  let commit;
  if (action.kind === "merge") {
    const repository = optionValue(action.tokens.slice(3), ["--repo", "-R"]);
    if (repository !== "AkumaRealLabs/ggapi") {
      return { violation: "PR merges must explicitly target AkumaRealLabs/ggapi" };
    }
    commit = optionValue(action.tokens.slice(3), ["--match-head-commit"]);
  } else {
    const endpoint = action.tokens.find((token) => /(?:^|\/)pulls\/[^/]+\/merge$/.test(token)) || "";
    if (!/^repos\/AkumaRealLabs\/ggapi\/pulls\/[^/]+\/merge$/.test(endpoint.replace(/^\//, ""))) {
      return { violation: "REST PR merges must explicitly target AkumaRealLabs/ggapi" };
    }
    const field = action.tokens.find((token) => /^(?:sha=|sha:)/.test(token));
    commit = field ? field.replace(/^sha[=:]/, "") : null;
  }
  if (!commit) return { violation: "PR merges must pin the reviewed head commit" };
  return { tree: runGit(root, ["rev-parse", "--verify", `${commit}^{tree}`]) };
}

function tagTarget(root, action) {
  if (path.basename(action.tokens[0] || "") === "gh") {
    if (action.tokens[1] === "release") {
      if (action.tokens[2] !== "create") return null;
      return optionValue(action.tokens.slice(3), ["--target"]) || "HEAD";
    }
    // gh api .../git/refs -f sha=<oid>
    const sha = action.tokens
      .map((token) => /^(?:sha|ref)=(.+)$/.exec(token)?.[1])
      .find((value) => value && /^[0-9a-f]{7,40}$/i.test(value));
    return sha || null;
  }
  const valueOptions = new Set(["-m", "--message", "-F", "--file", "--cleanup", "--format", "--sort", "--contains", "--points-at", "--column"]);
  const positional = [];
  for (let index = 2; index < action.tokens.length; index += 1) {
    const token = action.tokens[index];
    if (valueOptions.has(token)) index += 1;
    else if (!token.startsWith("-") && !["delete", "list", "verify"].includes(token)) positional.push(token);
  }
  if (hasFlag(action.tokens, ["-d", "--delete"])) return null;
  return positional[1] || "HEAD";
}

function resolvesToTag(root, ref) {
  if (ref.startsWith("refs/tags/")) return true;
  return spawnSync("git", ["rev-parse", "--verify", "--quiet", `refs/tags/${ref}`], { cwd: root, encoding: "utf8" }).status === 0;
}

function resolvesToBranch(root, ref) {
  if (ref.startsWith("refs/heads/")) return true;
  if (ref.startsWith("refs/tags/")) return false;
  return ["refs/heads", "refs/remotes/origin"].some((prefix) =>
    spawnSync("git", ["rev-parse", "--verify", "--quiet", `${prefix}/${ref}`], { cwd: root, encoding: "utf8" }).status === 0);
}

// git accepts several spellings of the same ref: main, heads/main and
// refs/heads/main all update origin/main.
function targetsMainBranch(ref) {
  return ["main", "heads/main", "refs/heads/main"].includes(ref.replace(/^\^?\+/, ""));
}

function gitInvocationViolation(root, action) {
  const options = action.gitOptions || {};
  if (options.overridesConfig || options.overridesRepository) {
    return "Ship mutations cannot override Git configuration or repository paths";
  }
  let directory = root;
  for (const value of options.directories || []) directory = path.resolve(directory, value);
  if (findRepositoryRoot(directory) !== root) {
    return "Ship mutations must target the current ggapi repository";
  }
  return null;
}

export function evaluateShipMutation(root, action, base = "origin/main") {
  const branch = currentBranch(root);
  if (action.kind === "unknown" || action.compound) {
    return { allowed: false, reason: "Ship mutations must run as one directly inspectable command" };
  }
  const gitViolation = gitInvocationViolation(root, action);
  if (gitViolation) return { allowed: false, reason: gitViolation };
  if (action.kind === "commit" && branch === "main") {
    return { allowed: false, reason: "Project policy forbids committing directly on main" };
  }
  if (action.kind === "push") {
    const pushed = pushedTree(root, action, branch);
    if (pushed.violation) return { allowed: false, reason: pushed.violation };
    if (pushed.deletion) return { allowed: true, reason: "explicit non-main remote branch deletion" };
    if (!pushed.tree) return { allowed: true, reason: "origin/main release tag" };
    const status = reviewGateStatusForTree(root, pushed.tree, base, branch);
    if (status.valid) return { allowed: true, reason: status.state, status };
    return invalidGateDecision(status);
  }
  if (action.kind === "tag") {
    const head = runGit(root, ["rev-parse", "HEAD"]);
    const main = runGit(root, ["rev-parse", "origin/main"]);
    // Deleting a LOCAL tag touches no remote ref and is a documented cleanup
    // step (SOP §5). Remote deletion still goes through push/gh and is gated.
    const deletesLocalTag = path.basename(action.tokens[0] || "") === "git"
      && hasFlag(action.tokens, ["-d", "--delete"]);
    if (deletesLocalTag) return { allowed: true, reason: "local tag deletion" };

    const target = tagTarget(root, action);
    // An unreadable target must not fall back to "assume origin/main" — that
    // turns the tip check into a vacuous pass.
    if (target === null) {
      return { allowed: false, reason: "Release tag target could not be resolved, so the origin/main tip rule cannot be checked" };
    }
    const targetCommit = runGit(root, ["rev-parse", "--verify", `${target}^{commit}`]);
    if (branch !== "main" || head !== main || targetCommit !== main) {
      return { allowed: false, reason: "Release tags must point at the origin/main tip" };
    }
  }
  if (action.temporarySnapshot) return { allowed: true, reason: "temporary review snapshot" };

  if (action.kind === "merge" || action.kind === "merge-rest") {
    const merged = mergeTree(root, action);
    if (merged.violation) return { allowed: false, reason: merged.violation };
    const status = reviewGateStatusForTree(root, merged.tree, base, branch);
    if (status.valid) return { allowed: true, reason: status.state, status };
    return invalidGateDecision(status);
  }

  const status = reviewGateStatus(root, base);
  if (!status.valid) return invalidGateDecision(status);

  // -a may be bundled (-am), in which case the commit stages the worktree and
  // the index comparison below does not apply. hasFlag so that "-m -a" — where
  // -a is the MESSAGE — does not skip the comparison.
  const stagesWorktree = hasFlag(action.tokens, ["--all"])
    || hasFlag(action.tokens, action.tokens.filter((token) => /^-[A-Za-z]*a[A-Za-z]*$/.test(token)));
  if (action.kind === "commit" && !stagesWorktree) {
    // "git commit" writes the index, which can hold content the worktree no
    // longer has. The reviewed worktree tree alone would miss that — including
    // on an "empty" (tree == base) branch, where the index is the only carrier.
    if (stagedTree(root) !== status.tree) {
      return { allowed: false, reason: "The staged index does not match the reviewed final tree", status };
    }
  }
  return { allowed: true, reason: status.state, status };
}

function invalidGateDecision(status) {
  return {
    allowed: false,
    reason: status.state === "stale"
      ? "The recorded Codex review is stale because the final tree or origin/main changed"
      : "The current final tree has not passed the Codex review gate",
    status,
  };
}
