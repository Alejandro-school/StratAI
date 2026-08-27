# StratAI agent guide

## Canonical context

- Use `README.md` for setup and commands.
- Use `docs/AI_COACH_PRODUCT_DEFINITION.md` for the product promise, scope, and non-negotiable behavior.
- Use `docs/AI_COACH_IMPLEMENTATION_PLAN.md` as the only source for delivery status, execution order, data gates, the next action, and the roadmap.
- Use `docs/AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md` for the human-readable capability and data contract.
- Use `ai_coach/contracts/capability_catalog.json` for stable machine-readable IDs, fields, actions, outcomes, and references.
- Use `docs/canonical-export-schema-v3.8.md` for the current bundle contract.
- Treat model architecture, dated audits, evidence, and archived contracts as references. They do not authorize work or replace the implementation plan. `docs/prompts/` contains only the next executable prompt; completed prompts are removed after their result is transferred to the plan and closure report.
- While `TEMPORARY_INTERFACE_MODE.txt` exists, preserve the temporary interface flow it documents.
- `.agents/skills` is the only repository-level skill catalog. Do not mirror it under `.agent` or `.github/skills`.

## Architecture

- `frontend/`: React 18 JavaScript application built with Vite.
- `backend/app/`: FastAPI web API and Steam authentication.
- `backend/node-service/`: Steam bot, demo acquisition, and queue orchestration.
- `backend/go-service/`: CS2 demo parsing and analysis.
- `backend/data/`: local/runtime data. Never treat it as disposable source code.

Keep domain logic out of route and UI layers when practical, and preserve the contracts between the four services.

## Code discovery

Prefer the codebase-memory graph for structural discovery:

1. `search_graph`
2. `trace_path`
3. `get_code_snippet`
4. `query_graph`
5. `get_architecture`

Use text search for literals, configuration, documentation, or gaps reported by the graph index.

## Change discipline

- Preserve existing user changes in a dirty worktree.
- Do not commit secrets, runtime databases, demos, exports, browser profiles, caches, compiled binaries, or dependency directories.
- Do not introduce Next.js, TypeScript, MUI, Expo, React Native, Remotion, or Better Auth conventions unless the project explicitly adopts them.
- Prefer focused feature modules and existing project abstractions over new generic helpers.

## Verification

Use the root scripts as the public verification interface:

- `npm run lint:python`
- `npm run test:python`
- `npm run test:node`
- `npm run test:go`
- `npm run test:frontend`
- `npm run build:frontend`
- `npm run check:all` for the full suite
