---
name: vercel-react-best-practices
description: Diagnose React render, request and bundle bottlenecks in StratAI. Use for frontend performance work or substantial React refactors.
license: MIT
metadata:
  author: Vercel Engineering; adapted for StratAI
  upstream-ref: 063bee94c3f4df8453406c830b0a7df0f2860278
---

# React performance for StratAI

Selected rules from [Vercel's official collection](https://github.com/vercel-labs/agent-skills/tree/063bee94c3f4df8453406c830b0a7df0f2860278/skills/react-best-practices). Read only the references relevant to the measured problem.

## Apply to this stack

StratAI uses React 18, JavaScript, Vite, TanStack Query and Zustand. Check the current package manifests before relying on version-specific APIs. Upstream TypeScript examples illustrate patterns; implement them in the existing JavaScript style. This selection does not introduce Next.js, server components/actions, SWR, React Compiler, Activity or useEffectEvent.

Start with a reproducible interaction and a baseline: render commits, request count/waterfalls, frame time or build chunk size. Prioritise the measured bottleneck. Upstream impact labels and speedup examples are illustrative, not measurements of StratAI.

## Load the relevant rule

| Problem | Reference |
| --- | --- |
| Independent requests form a waterfall | [Parallel operations](rules/async-parallel.md) |
| Touch or wheel listeners delay scrolling | [Passive listeners](rules/client-passive-event-listeners.md) |
| An effect mirrors props or state | [Derived state](rules/rerender-derived-state-no-effect.md) |
| A callback uses stale state | [Functional updates](rules/rerender-functional-setstate.md) |
| Initialisation repeats expensive work | [Lazy state](rules/rerender-lazy-state-init.md) |
| A measured expensive child rerenders | [Memoised components](rules/rerender-memo.md) |
| Memoisation costs more than the expression | [Simple expressions](rules/rerender-simple-expression-in-memo.md) |
| Frame or pointer values cause React renders | [Transient refs](rules/rerender-use-ref-transient-values.md) |
| Inputs lose focus or children remount | [Component identity](rules/rerender-no-inline-components.md) |
| Repeated joins scan the same array | [Index maps](rules/js-index-maps.md) |

Preserve semantics when applying these rules: parallelise independent reads, not ordered mutations; keep state that drives visible UI reactive; keep initialisers pure for StrictMode; define duplicate-key behaviour before replacing a find with a Map. Memoisation requires correct dependencies and evidence of benefit.

For bundle loading, use native import() or [React.lazy and Suspense](https://react.dev/reference/react/lazy) with the current Vite structure. Keep loading and error fallbacks. For server state, preserve existing [TanStack Query keys](https://tanstack.com/query/v5/docs/framework/react/guides/query-keys), invalidation, cancellation and user boundaries. Share listeners through existing owners/hooks where appropriate instead of adding another state library.

After a change, compare the same interaction and run the relevant frontend tests and build through the root scripts. Check replay seeking, speed and selected-player behaviour when they are affected. Record the actual improvement or state that the result remains unmeasured.
