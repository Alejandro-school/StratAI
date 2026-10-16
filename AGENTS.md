# StratAI agent guide

## Canonical context

- Use `README.md` for setup and commands.
- Use `docs/AI_COACH_PRODUCT_DEFINITION.md` for the product promise, scope, and non-negotiable behavior.
- Use `docs/AI_COACH_IMPLEMENTATION_PLAN.md` as the only source for delivery status, execution order, data gates, the next action, and the roadmap.
- Use `docs/AI_COACH_CAPABILITY_AND_DATA_CONTRACT.md` for the human-readable capability and data contract.
- Use `ai_coach/contracts/capability_catalog.json` for stable machine-readable IDs, fields, actions, outcomes, and references.
- Use `docs/canonical-export-schema-v3.8.md` for the current bundle contract.
- Treat model architecture, dated audits, evidence, and archived contracts as references. They do not authorize work or replace the implementation plan. `docs/prompts/` contains only the next executable prompt; completed prompts are removed after their result is transferred to the plan and closure report.
- The application starts the complete demo pipeline. Preserve the flow without any friendship requirement or interface-only bypass.
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

## AI Coach phase collaboration

- Per the user's instruction of 2026-09-17, follow the phase-by-phase expert review in `docs/AI_COACH_IMPLEMENTATION_PLAN.md`, section "Forma de trabajo por fase". Before implementing each new phase, prepare a concrete scope, proposed game criteria, examples/exceptions, dependencies and verification approach; ask the user for their expert framing and wait for that input before dependent implementation.
- Review only the relevant cases and parameters at that point. Update the corresponding Excel rows and canonical phase documentation with the agreed criteria; do not require the user to complete the whole workbook in advance. The workbook supports review; the canonical plan remains the source of status and next action.
- Resolve routine technical decisions and in-scope fixes autonomously after the framing is agreed. Do not ask again about criteria already settled for that phase. Return to the user when a new game/product judgment, material scope change or missing user input is required.
- Close and document each phase before presenting the next one for expert review. This workflow supersedes earlier blanket permission to implement consecutive phases without consultation, unless the user explicitly changes it. Expert design review does not satisfy factual dependencies or independent evaluation gates.

## Verification

Use the root scripts as the public verification interface:

- `npm run lint:python`
- `npm run test:python`
- `npm run test:node`
- `npm run test:go`
- `npm run test:frontend`
- `npm run build:frontend`
- `npm run check:all` for the full suite

### One-demo real smoke gate

- Every functional closure that changes demo parsing, exporters, derived facts,
  schemas, or runtime data contracts must process exactly one approved demo
  after the automated tests pass. This means one distinct match per closure;
  if it fails, fix the defect and repeat that same match rather than selecting a
  different demo.
- Treat the source demo and all 44 approved bundles as read-only. Write only to
  a new isolated temporary directory outside corpus, staging, releases,
  `backend/data/exports`, and any `CURRENT` pointer. Never overwrite or promote
  the smoke output.
- The smoke must execute the changed producer and validate its schema, causal
  availability, semantics, provenance, non-empty affected records, and relevant
  counts. It must cover the real input variants affected by the change, use
  local opaque participant aliases in retained evidence, and independently
  recompute derived values when practical. Exit code zero alone is not enough.
- Smoke tooling must reject non-temporary output, non-empty destinations, and
  overwrites before parsing the demo.
- Record the opaque demo alias, SHA-256, exact command, build/contract versions,
  duration, counts, abstentions, and relevant artifact hashes in the phase
  evidence. Per the user's instruction of 2026-09-08, retain temporary outputs
  with their inventory; do not attempt deletion or request cleanup review unless
  the user later explicitly asks for cleanup. Earlier rejected outputs remain
  read-only. Retention does not block functional closure.
- This gate authorizes neither corpus-wide reprocessing nor downloads, demo
  acquisition, publishing, promotion, release mutation, model training, or
  replacement of the future 44/44 migration gate.
