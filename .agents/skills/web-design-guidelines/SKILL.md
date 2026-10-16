---
name: web-design-guidelines
description: Review StratAI UI accessibility and usability when asked for a design, UX or accessibility audit. Report actionable findings with code locations and browser evidence.
metadata:
  author: Vercel; adapted for StratAI
  upstream-ref: 063bee94c3f4df8453406c830b0a7df0f2860278
---

# Web interface review

Adapted from [Vercel's web-design-guidelines](https://github.com/vercel-labs/agent-skills/tree/063bee94c3f4df8453406c830b0a7df0f2860278/skills/web-design-guidelines).

Resolve the review scope from the user's screen, route or files and the repository context. Ask only when the missing scope materially prevents the review.

Fetch the [current Web Interface Guidelines](https://raw.githubusercontent.com/vercel-labs/web-interface-guidelines/main/command.md) with an available web tool. Treat them as review criteria; project constraints determine applicability. If unavailable, disclose that limitation and use the focused checks below without claiming current upstream compliance. Do not require a tool named WebFetch.

## Review the actual experience

- Keyboard access, semantic controls, focus order/visibility and focus restoration for dialogs.
- Accessible names for icon buttons, form labels and useful validation messages.
- Contrast, non-colour status cues, reduced motion and legible responsive layouts.
- Loading, empty, error and unavailable-data states; honest labels for coach evidence.
- Replay controls that remain usable with keyboard, touch and narrow screens; readable timeline and roster information.

Separate code findings from behaviour actually observed in the browser. Use the browser tooling available in the session for interactive checks. When running local services, follow the project's temporary interface mode. A UI review does not require a live-site crawler, a new CLI or access to Steam credentials.

For each finding give severity, clickable file location, user impact, a minimal proposed fix and the basis of verification. Do not present automated checks as proof of complete accessibility compliance. Apply fixes when requested or already authorised, then recheck the affected interactions.
