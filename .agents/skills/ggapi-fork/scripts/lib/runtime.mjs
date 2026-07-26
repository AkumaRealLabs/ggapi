import fs from "node:fs";
import path from "node:path";

const SURFACES = new Set(["codex", "grok", "claude", "generic"]);

function commandExists(command, env = process.env) {
  const pathValue = env.PATH || "";
  const extensions = process.platform === "win32"
    ? (env.PATHEXT || ".EXE;.CMD;.BAT;.COM").split(";")
    : [""];

  return pathValue.split(path.delimiter).some((directory) => {
    if (!directory) return false;
    return extensions.some((extension) => {
      try {
        fs.accessSync(path.join(directory, `${command}${extension}`), fs.constants.X_OK);
        return true;
      } catch {
        return false;
      }
    });
  });
}

function parentProcessNames() {
  if (process.platform !== "linux") return [];

  const names = [];
  let pid = process.ppid;
  for (let depth = 0; depth < 6 && pid > 1; depth += 1) {
    try {
      const commandLine = fs.readFileSync(`/proc/${pid}/cmdline`, "utf8");
      names.push(commandLine.replaceAll("\0", " ").toLowerCase());
      const stat = fs.readFileSync(`/proc/${pid}/stat`, "utf8");
      const closingParen = stat.lastIndexOf(")");
      pid = Number(stat.slice(closingParen + 2).split(" ", 2)[1]);
    } catch {
      break;
    }
  }
  return names;
}

export function detectSurface({ explicit = "auto", env = process.env, hookInput = {} } = {}) {
  if (explicit !== "auto") {
    if (!SURFACES.has(explicit)) throw new Error(`Unsupported agent surface: ${explicit}`);
    return explicit;
  }

  if (env.GROK_HOOK_EVENT || env.GROK_SESSION_ID) return "grok";
  if (env.CLAUDECODE || env.CLAUDE_CODE_ENTRYPOINT) return "claude";
  if (env.CODEX_THREAD_ID || hookInput.turn_id || hookInput.turnId) return "codex";
  if (env.CLAUDE_PROJECT_DIR) return "claude";

  for (const commandLine of parentProcessNames()) {
    if (/(^|[/ ])codex([ .]|$)/.test(commandLine)) return "codex";
    if (/(^|[/ ])grok([ .]|$)/.test(commandLine)) return "grok";
    if (/(^|[/ ])claude([ .]|$)/.test(commandLine)) return "claude";
  }
  return "generic";
}

// Headless review instruction for surfaces without a dedicated review
// subcommand. "origin/main" is substituted with the requested base by
// formatReviewCommand.
const HEADLESS_REVIEW_PROMPT =
  "Perform a strict code review of the combined diff from origin/main to the "
  + "current working tree, including uncommitted files. Report only real "
  + "defects with file and line references. Do not modify any files.";

// Each surface reviews with its OWN reviewer. A surface whose reviewer is
// installed but failing (rate limit, auth) is a BLOCKED gate, not a licence to
// substitute another vendor's reviewer — see troubleshooting §21. Only a
// surface that has no reviewer of its own (generic) may use the Codex CLI.
export function selectReviewStrategy(surface, { env = process.env } = {}) {
  if (surface === "claude") {
    return commandExists("claude", env)
      ? { kind: "claude-native", command: "claude", args: ["-p", "/code-review origin/main"], reviewer: "claude", fallback: null }
      : { kind: "claude-in-session", command: null, args: [], reviewer: "claude", fallback: null };
  }

  if (surface === "grok") {
    return commandExists("grok", env)
      ? { kind: "grok-native", command: "grok", args: ["-p", HEADLESS_REVIEW_PROMPT], reviewer: "grok", fallback: null }
      : { kind: "grok-in-session", command: null, args: [], reviewer: "grok", fallback: null };
  }

  if (commandExists("codex", env)) {
    return {
      kind: surface === "codex" ? "codex-native" : "codex-cli-fallback",
      command: "codex",
      args: ["review", "--base", "origin/main"],
      reviewer: "codex",
      fallback: null,
    };
  }

  // Symmetric with Claude and Grok: a surface can always review with its own
  // capability, so a missing CLI is not a hard gate block.
  if (surface === "codex") {
    return { kind: "codex-in-session", command: null, args: [], reviewer: "codex", fallback: null };
  }

  return {
    kind: "unavailable",
    command: null,
    args: [],
    reviewer: null,
    fallback: null,
  };
}

export function describeRuntime({ explicit = "auto", env = process.env, hookInput = {}, root }) {
  const surface = detectSurface({ explicit, env, hookInput });
  const review = selectReviewStrategy(surface, { env });
  const projectRoot = root || process.cwd();

  return {
    surface,
    instructions: surface === "claude" ? "CLAUDE.md -> AGENTS.md" : "AGENTS.md",
    skills: fs.existsSync(path.join(projectRoot, ".agents", "skills")),
    hooks: surface === "codex"
      ? fs.existsSync(path.join(projectRoot, ".codex", "hooks.json"))
      : fs.existsSync(path.join(projectRoot, ".claude", "settings.json")),
    review,
  };
}

function shellQuote(value) {
  if (/^[A-Za-z0-9_./:@=-]+$/.test(value)) return value;
  return `'${value.replaceAll("'", `'"'"'`)}'`;
}

export function formatReviewCommand(strategy, base = "origin/main") {
  if (!strategy.command) return null;
  const format = ({ command, args }) => {
    const resolvedArgs = args.map((argument) => argument.replaceAll("origin/main", base));
    return [command, ...resolvedArgs].map(shellQuote).join(" ");
  };
  const primary = format(strategy);
  return strategy.fallback ? `${primary} || ${format(strategy.fallback)}` : primary;
}
