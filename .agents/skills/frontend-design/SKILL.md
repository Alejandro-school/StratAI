---
name: frontend-design
description: Design or restyle StratAI screens and components with a coherent visual direction. Use for visual UI work, not backend changes or routine logic fixes.
license: Complete terms in LICENSE.txt
metadata:
  author: Anthropic; adapted for StratAI
  upstream-ref: 683bc88e56f3e09ba94f7055977f3d3aa499f202
---

# Frontend design for StratAI

Adapted from [Anthropic frontend-design](https://github.com/anthropics/skills/tree/683bc88e56f3e09ba94f7055977f3d3aa499f202/skills/frontend-design).

## Fit the existing product

Inspect the affected screen, its styles, and neighbouring components before choosing a visual direction. Preserve the established typography, navigation, tokens and component patterns unless the brief calls for a redesign. StratAI is a CS2 analysis product; prioritise readable statistics, tactical context and actionable coaching. Consult the product definition for claims and feature scope.

Use the existing React 18, JavaScript, Vite and CSS stack. Use installed icons and motion libraries when they help. A small UI edit does not require a new design system or dependency.

## Design and build

- Identify the main user action and information hierarchy. For a redesign, state a brief visual direction grounded in the screen's purpose; for a small edit, follow the existing direction.
- Use deliberate type scale, contrast and spacing. Reuse project fonts; avoid arbitrary font bans and decorative changes that reduce readability.
- Choose structure by content: tables for comparable statistics, grouped controls for replay, clear hierarchy for coaching. Do not force every item into an identical card.
- Reserve motion for explaining interaction or drawing attention to a meaningful change. Respect reduced motion and protect replay responsiveness.
- Write concise Spanish interface copy consistent with the surrounding UI. Keep loading, empty, error and unavailable-data states distinct. Do not invent statistics or present placeholder coaching as demonstrated capability.
- Keep keyboard focus visible, controls labelled, and essential information available beyond colour or hover alone. On narrow screens preserve access to the primary action.

## Verify

Inspect the result in the browser at desktop and narrow widths when the environment permits, including the affected interaction and its empty/error states. Report any visual check that could not be performed. Use the repository frontend checks for code changes; follow the temporary interface flow documented by the project when starting services.
