# web/ggapi — full paper-sketch redesign

Source: **tech-visual-explainer v1.0.0** sketch tokens + layout habits.
Scope: **entire product shell**, not a homepage skin on stock new-api chrome.

## Product identity

| Surface | Behavior |
|---------|----------|
| Default name | `ggapi` (`DEFAULT_SYSTEM_NAME`) |
| Default theme preset | `paper-sketch` + font `hand` |
| Protected attribution | Footer still credits **New API / QuantumNous** (do not remove) |

## Architecture of the redesign

```
src/styles/theme-presets.css   → color tokens (paper / chalk)
src/styles/paper-sketch.css    → FULL shell chrome (buttons, tables, sidebar, auth, header…)
src/features/home/**           → marketing home rewritten as explainer page
src/features/auth/auth-layout  → notebook auth card
src/components/layout/**       → public header / footer / app header hooks
src/components/ui/button.tsx   → data-variant for CSS targeting
```

## Visual rules (non-negotiable)

1. **Ink + paper only** — derive with `color-mix`; no new random gradients/glows.
2. **Wobble borders** on cards/panels/primary chrome (`--wobble-a` / `--wobble-b`).
3. **No glass SaaS** — no floating blurred pill nav; notebook strip header; no soft `shadow-*` / `backdrop-blur` in console.
4. **Motion** only for state / tiny hover lift; honor `prefers-reduced-motion`.
5. **Console is in scope** — admin dashboard, tables, sidebar, dialogs, section titles all use the same paper-sketch chrome as the home explainer (not a thin tint on stock new-api UI).
6. Template token aliases (`--blue` / `--teal` / `--amber` / `--purple` / `--red` / `--ease`) mirror design-system.md names.

## How to verify

```bash
make dev-web-ggapi
# Hard refresh http://localhost:5175
# Clear localStorage.status if name stuck on "New API"
# Walk: home → pricing → sign-in → dashboard → settings
```

## Feature tracking

Keep porting **behavior** from `web/default` into this shell; do **not** reintroduce the stock SaaS landing or glass chrome.
