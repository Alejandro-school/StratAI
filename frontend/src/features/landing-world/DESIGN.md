---
name: StratAI Mirage official landing
description: Feature-local visual reference for the official StratAI root landing at /.
colors:
  mint: "#b1f1d8"
  mint-hover: "#d8ffed"
  primary-ink: "#112b26"
  action-ink: "#102720"
  replay: "#a8efd5"
  utility: "#ffbe7b"
  stats: "#c1c5ff"
  coach: "#94d8ff"
  challenges: "#f3ca89"
  ink: "#edf2e8"
  description: "#bdcfc0"
  content-description: "#bcd1bf"
  disclosure: "#afc6ba"
  capability: "#c8dfcf"
  media-caption: "#c8dfd0"
  media-prompt: "#c1d6c5"
  demo-chapter: "#c5ddcb"
  demo-insight: "#c6d9c9"
  demo-error: "#f6c4aa"
  replay-player: "#ef4444"
  replay-player-label: "#fca5a5"
  replay-player-label-bg: "#ef44444d"
  replay-player-outline: "#ef444480"
  replay-teammate: "#eab308"
  replay-teammate-label: "#fde047"
  replay-teammate-label-bg: "#eab3084d"
  replay-teammate-outline: "#eab30880"
  question: "#d6e5d4"
  outcome: "#c9dbca"
  plan-feature: "#d3e2d1"
  footer-action: "#d9e8d8"
  world: "#0b191c"
  panel: "#0b2023fc"
  media: "#07171b"
  media-caption-bg: "#153034"
  demo-coach: "#10292c"
  dock: "#102325ee"
  pin: "#0e2426ed"
  pin-selected: "#24433ef5"
  line: "#b0c6b82a"
  panel-border: "#819d8f77"
  dock-border: "#89a79766"
  link-border: "#bed6c966"
  plan-border: "#72998a66"
  plan-divider: "#72998a44"
  flight-description: "#d1ded2"
  camera-guidance: "#e0eadb"
  camera-hint: "#c6d8c7"
  camera-tools: "#102325f5"
  camera-hover: "#30503f"
typography:
  display:
    fontFamily: "Rajdhani, sans-serif"
    fontSize: "clamp(52px, 4.8vw, 72px)"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "-0.025em"
  headline:
    fontFamily: "Rajdhani, sans-serif"
    fontSize: "42px"
    fontWeight: 600
    lineHeight: 1.06
    letterSpacing: "-0.02em"
  title:
    fontFamily: "Rajdhani, sans-serif"
    fontSize: "25px"
    fontWeight: 600
    lineHeight: 1.15
  body:
    fontFamily: "Inter, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.8
  replay-body:
    fontFamily: "Inter, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.85
  replay-marker:
    fontFamily: "Inter, sans-serif"
    fontSize: "10.4px"
    fontWeight: 700
    letterSpacing: ".05em"
  replay-clock:
    fontFamily: "Rajdhani, sans-serif"
    fontSize: "18px"
  replay-coach-title:
    fontFamily: "Rajdhani, sans-serif"
    fontSize: "23px"
    fontWeight: 600
  capability:
    fontFamily: "Inter, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.6
  supporting:
    fontFamily: "Inter, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.85
  label:
    fontFamily: "Inter, sans-serif"
    fontSize: "11px"
    fontWeight: 400
  micro:
    fontFamily: "Inter, sans-serif"
    fontSize: "10px"
    fontWeight: 400
  hero-body:
    fontFamily: "Inter, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.85
  plan-price:
    fontFamily: "Rajdhani, sans-serif"
    fontSize: "37px"
    fontWeight: 600
    lineHeight: 1.1
  currency:
    fontFamily: "Rajdhani, sans-serif"
    fontSize: "21px"
  brand:
    fontFamily: "Inter, sans-serif"
    fontSize: "28px"
    fontWeight: 800
    letterSpacing: "-1.2px"
  flight:
    fontFamily: "Rajdhani, sans-serif"
    fontSize: "clamp(38px, 4vw, 60px)"
    fontWeight: 600
    lineHeight: 1.1
  steam-entry-brand:
    fontFamily: "Inter, sans-serif"
    fontSize: "clamp(42px, 5vw, 64px)"
    fontWeight: 800
    lineHeight: 1
    letterSpacing: "-.035em"
  steam-entry-status:
    fontFamily: "Inter, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.8
  camera-guidance:
    fontFamily: "Inter, sans-serif"
    fontSize: "12px"
    fontWeight: 400
    lineHeight: 1.8
rounded:
  replay-marker-label: "12px"
  motion-track: "10px"
  scroll-hint: "15px"
  dot: "50%"
spacing:
  control-gap: "10px"
  panel-body: "28px 32px"
  panel-body-mobile: "22px 20px"
  panel-body-narrow: "22px 20px"
  replay-body: "20px 26px"
  replay-marker-label: "4px 10px"
  desktop-edge: "44px"
  steam-entry-mark: "24px"
components:
  button-primary:
    backgroundColor: "{colors.mint}"
    textColor: "{colors.primary-ink}"
    padding: "0 20px"
  button-primary-hover:
    backgroundColor: "{colors.mint-hover}"
  header-explore:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    padding: "12px 16px"
  product-navigation:
    backgroundColor: "{colors.dock}"
  product-navigation-button:
    backgroundColor: "transparent"
    textColor: "#d5e4d4"
    padding: "15px 18px"
  world-pin:
    backgroundColor: "{colors.pin}"
    textColor: "#edf3e8"
    padding: "0 13px"
    width: "170px"
  world-pin-selected:
    backgroundColor: "{colors.pin-selected}"
  feature-panel:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    width: "min(1080px, 68vw)"
  replay-panel:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    width: "calc(100% - 96px)"
  replay-marker:
    rounded: "{rounded.dot}"
    width: "60px"
    height: "60px"
  replay-player-label:
    backgroundColor: "{colors.replay-player-label-bg}"
    textColor: "{colors.replay-player-label}"
    typography: "{typography.replay-marker}"
    rounded: "{rounded.replay-marker-label}"
    padding: "{spacing.replay-marker-label}"
  replay-teammate-label:
    backgroundColor: "{colors.replay-teammate-label-bg}"
    textColor: "{colors.replay-teammate-label}"
    typography: "{typography.replay-marker}"
    rounded: "{rounded.replay-marker-label}"
    padding: "{spacing.replay-marker-label}"
  replay-coach:
    backgroundColor: "{colors.demo-coach}"
    textColor: "{colors.question}"
  feature-body:
    padding: "{spacing.panel-body}"
  panel-action:
    backgroundColor: "var(--feature-color)"
    textColor: "{colors.action-ink}"
    padding: "10px 13px"
  screenshot:
    backgroundColor: "{colors.media}"
    width: "100%"
  screenshot-caption:
    backgroundColor: "{colors.media-caption-bg}"
    textColor: "{colors.media-caption}"
    padding: "12px 14px"
  screenshot-dialog:
    backgroundColor: "#0b2023"
    textColor: "{colors.ink}"
    width: "min(1300px, calc(100vw - 50px))"
  information-tab:
    backgroundColor: "transparent"
    textColor: "#b9d0be"
    padding: "10px 12px"
  billing-selected:
    backgroundColor: "{colors.mint}"
    textColor: "{colors.action-ink}"
    padding: "11px 20px"
  development-status:
    backgroundColor: "#80b7dd17"
    textColor: "#a9d9ff"
    padding: "9px 11px"
  steam-entry:
    backgroundColor: "{colors.world}"
    textColor: "{colors.ink}"
  steam-entry-mark:
    padding: "{spacing.steam-entry-mark}"
  steam-entry-brand:
    textColor: "{colors.ink}"
    typography: "{typography.steam-entry-brand}"
  steam-entry-status:
    textColor: "{colors.media-caption}"
    typography: "{typography.steam-entry-status}"
  camera-tools:
    backgroundColor: "{colors.camera-tools}"
  camera-tool:
    backgroundColor: "transparent"
    textColor: "#d5e4d4"
    padding: "10px 13px"
  camera-tool-hover:
    backgroundColor: "{colors.camera-hover}"
    textColor: "{colors.mint}"
---

# Design System: StratAI Mirage official landing

## Overview

**Creative North Star: "Entrar en la ronda"**

This reference records the official StratAI landing at the public application route `/`, entered through `frontend/index.html` and the lazy feature import in `App.jsx`. It replaces the former root landing; `frontend/landing-world.html` remains an optional standalone entry. Native Mirage geometry, UVs and textures provide warm stone, blue and salmon facades against a deep teal horizon with web lighting. Compact dark navigation leaves the map visible; opening a tool gives its content a broad reading surface. Rajdhani retains the condensed game identity; Inter carries the original Spanish landing copy and product information.

The direction and composition remain in [SURFACE.md](SURFACE.md). This reference governs the landing feature, without extending its styles to authenticated application screens. The official root integration is a source change, not a remote deployment. The [canonical product definition](../../../../docs/AI_COACH_PRODUCT_DEFINITION.md) governs product promises; the [implementation plan](../../../../docs/AI_COACH_IMPLEMENTATION_PLAN.md) governs delivery status. Screenshots and the scripted demo do not establish automatic coaching availability or production readiness.

**Key Characteristics:**

- Native textured Mirage and web lighting provide real scene depth.
- Five labeled destinations also remain available in the bottom dock.
- Larger tool panels reuse original Spanish copy and screenshots; a wide replay pairs the original Dust2 video with its scripted, progressive Coach commentary.
- A static CT on A and T on stairs ground the final open camera view.
- Button-led flight hands over to bounded pan, rotation and zoom in native-detail A.
- Demand rendering restores the initial sharp pixel ratio after movement settles.
- Tool changes coordinate a short camera arc with panel exit and entry; Steam sign-in departs through the existing Mirage scene before the existing authentication handoff.
- Coach IA remains in development and evaluation; challenges and plans remain proposals.

## Colors

Pale mineral text sits over dark teal with sparse mint guidance and five destination accents. Frontmatter records scoped source values; supporting tints preserve the actual media, guide, plan and FAQ treatments.

### Primary

- **Journey Mint** (`mint`, `mint-hover`): entry actions, wordmark accent, focus outlines, flight copy, camera-control hover, selected information tabs and billing period.
- **Primary and Action Ink** (`primary-ink`, `action-ink`): dark text on light accent actions.

### Secondary

- **Replay Mint**, **Map Amber**, **Statistics Lavender**, **Coach Sky** and **Challenges Sand** (`replay`, `utility`, `stats`, `coach`, `challenges`): their corresponding world destinations and panel controls.
- **Demo Error Peach** (`demo-error`): the existing scripted insight's error tone.
- **Original Player Red and Teammate Yellow** (`replay-player`, `replay-teammate`): the restored video annotation rings. Their corresponding label, label-background and outline tokens retain the source treatments. These colors belong to the original replay annotations, alongside the destination palette.

### Neutral

- **Mineral White and supporting minerals** (`ink`, `description`, `content-description`, `disclosure`): titles, descriptions and proposal notes.
- **Media, question and plan minerals** (`capability` through `footer-action`): small copy variations in their named contexts.
- **World, Panel, Dock and Pin Teals** (`world`, `panel`, `dock`, `pin`, `pin-selected`): scenery, reading surfaces and navigation. The panel is nearly opaque for long copy and media.
- **Media Teals** (`media`, `media-caption-bg`, `demo-coach`): contain images and video without cropping and distinguish the adjacent Coach reading column.
- **Quiet Dividers** (`line`, `panel-border`, `dock-border`, `link-border`, `plan-border`, `plan-divider`): chrome, tabs, links and plan columns.
- **Flight and Camera Minerals** (`flight-description`, `camera-guidance`, `camera-hint`): flight subtitle and the two levels of camera instructions. **Camera Teals** (`camera-tools`, `camera-hover`) extend the existing dock treatment to the compact camera controls.

Feature colors originate in `journey.js` and feed `--feature-color` and `--pin-color`. Basic variables live in `world.css`; other values are observed scoped literals. Unused `--muted` is excluded.

**The Destination Rule.** Preserve the five destination colors together with text labels, icons and explicit selection states.

**The Replay Marker Rule.** Preserve the original red player ring and yellow teammate ring and their labels; center them on the calibrated blue-ball positions at the exact 1.3s and 6s pause frames.

## Typography

**Display Font:** Rajdhani, with sans-serif fallback.  
**Body Font:** Inter, with sans-serif fallback.

Both families are requested through the official `frontend/index.html` Google Fonts stylesheet alongside the app's existing fonts. Condensed headings retain the game identity; larger Inter descriptions, capability lists and replay comments support sustained reading. Small copy remains for navigation and metadata. This is an observed hierarchy, not a mathematical scale.

### Hierarchy

- **Display:** “Tu partida, desde dentro.”, balanced two-line wrapping and mint second line.
- **Headline:** product/information panel titles, balanced wrapping and maximum width (24ch).
- **Title:** guide steps, demo subsections and plan names. Inside-map copy uses Rajdhani (38px), line-height (1).
- **Body:** product introductions use the larger `body` role; capability lists use `capability`. The hero uses `hero-body`. Supporting guide and FAQ explanations use `supporting`, with FAQ leading increasing to (1.9).
- **Replay Body, Clock and Coach Title:** commentary uses `replay-body`; the transport clock uses tabular Rajdhani numerals; the adjacent Coach heading uses `replay-coach-title`. The original annotation labels use `replay-marker`, with an Eye icon and compact, bold lettering. Its fixed pixel size preserves the former 16px-root appearance even when the host app uses a 14px mobile root.
- **Label and Micro:** navigation, captions, disclosures, dense plan details and panel end matter.
- **Plan Price:** tabular numerals, with separate currency and period sizing.
- **Brand:** wordmark with the AI segment in Journey Mint. The Steam departure mark uses `steam-entry-brand`, with its connection status in `steam-entry-status`.
- **Flight:** “Entrando en A” / “Volviendo al aire”, with a 13px Inter subtitle at line-height (1.8). Camera guidance uses 12px Inter at line-height (1.8), a 10px secondary instruction and 11px action labels.

Responsive type is intentional: display becomes (78px) at wide widths, (61px) at compact desktop widths and `clamp(43px, 8.5vw, 61px)` on mobile; the final exploration stylesheet sets narrow-phone display to (60px), including short narrow viewports. Other short mobile viewports use (40px). Generic panel titles become (46px), (38px), (36px) and (33px) across wide, compact, mobile and narrow layouts. Replay titles use (36px) at and below (1100px), then (34px) on mobile. Replay comments become (13px) at and below (1100px); mobile replay descriptions and capability lists also use (13px). Mobile subsection titles use (23px), inside heading (29px), camera guidance (11px), and mobile/narrow wordmarks (24px)/(23px). Plan prices use (32px) on compact desktops and (38px) on mobile.

## Layout

The feature root is fixed to the viewport, with its own (16px) base font and hidden overflow. A fixed full-viewport scene sits behind fixed header, footer and narrative overlays. Entry and return actions control travel. All feature selectors, resets, focus styles, scrollbars and reduced-motion rules are scoped under `.landing-world`; no global html/body scroll lock is installed, so retained styles do not affect later SPA routes after the landing unmounts. `index.js` owns the explicit order `world.css`, `content.css`, `exploration.css`, `productPresentation.css`, then `transitions.css`; both official and optional standalone entries use that feature module. Entry copy sits left (6%), top (23%), maximum width (560px), leaving the world open to the right. The desktop header is (88px) high and has no lower divider; the footer minimum height is (100px). Its current footer copy is “ANALIZA TU PARTIDA”; it is footer text, not a new global label category. Flight copy sits left (6%), top (29%); exploration controls sit left (44px), bottom (130px), with maximum width (390px). The current component mounts no scroll spacer, scroll hint or journey rail; their older stylesheet selectors are historical implementation residue.

Modes are `aerial`, `entering`, `exploring` and `leaving`. “Estoy listo” and the header entry action start a (3.4s) flight to the arrival camera `[-6, 17, 34]`, target `[3, 0, 26]`, framing stairs, A and CT boxes. The brand, header return and “Volver al aire” reverse travel. Feature actions can also enter A before opening their panel. In exploration, absolute target bounds are X (0–4.8), Z (22.5–26.8), distance (19–26), yaw offset from the original view (−0.2 to +0.06 radians), elevation offset (−0.05 to +0.08 radians) and lens zoom (0.8–2.1). Pan, yaw, elevation and spare dolly distance share one asymmetric ellipsoidal envelope; their independent extremes cannot combine. Distance starts at `17 / sin(elevation)` to keep the (17)-unit height floor, with a spare-dolly radius of (2.5). Actual combined movement is smaller than the absolute bounds. The retained real-map check samples (972) poses, including cursor offsets, with no camera below an opaque roof at a (0.6)-unit margin; this is a sampled check, not a general collision engine.

Opening a product destination temporarily uses its authored framing while keeping manual navigation values. Closing it restores that exploration view. Desktop destinations other than Mapa offset their target sideways by (3) scene units; mobile omits that offset. Mapa targets `[8, 2, 25]` and uses `PerspectiveCamera.setViewOffset` with half the responsive panel-plus-gutter occlusion as its horizontal offset, placing the arc and smoke in the remaining left scene area. The `utilityPanelOcclusion` helper in `sceneLayout.js` mirrors the panel widths and right gutters; it returns zero on mobile. Leaving Mapa, or entering its mobile layout, clears the projection offset with `clearViewOffset`.

Generic panels occupy top (109px), right (34px), bottom (118px), with the frontmatter width and no maximum-height cap. The body scrolls independently with stable scrollbar space and contained overscroll. Chrome, information navigation and footer remain outside the body. Screenshots can reach `min(55svh, 620px)` in height and retain their original proportions, with a larger native dialog.

Replay uses a centered panel with maximum width (1960px), left/right edges (48px), top (96px) and bottom (98px). Video and Coach sit side by side on desktop. The video width is capped by a height budget of `max(220px, 100svh - 535px)` and its original (16:9) frame; the Coach column has minimum width (300px) and maximum height `max(340px, 100svh - 451px)`. Its messages scroll independently. This preserves the calibrated annotation alignment within the 16:9 source frame while keeping the transport available. The static replay screenshot and original capability list remain reachable through native disclosure below the live demo.

Five callouts use projected world anchors and SVG leaders. Desktop placement tries separated slots, clamps horizontal centers between (100px) and (viewport width minus 100px), clears the introduction and reserves the footer. The camera guidance and controls reserve the full bottom-left area from `height - 270px`, through X (452px), or (468px) at wide widths; label bounds stay outside it. Coach prefers a (130px) downward displacement from its projected anchor to leave the posed T visible; other callouts prefer (65px), subject to the same clamp and collision checks. Mobile reserves the full width below `height - 270px` and adapts dedicated slots: Stats/Replay pair at `min(455px, height - 355px)`, Coach left at `min(315px, lowerRow - 62px)`, Mapa right at `min(245px, coachRow - 65px)`, and Retos centered at `min(530px, height - 307px)`. These rows preserve room for the static T and camera instructions on short narrow screens. Leaders retain real anchors. This layout is specific to these five callouts.

### Responsive behavior

- **At and above 1900px:** page and camera-control edges (60px), hero width (650px), top (25%), flight copy left (8%); generic panel width `min(1200px, 62vw)`, top (120px), right (60px). Generic body padding stays at the frontmatter value; replay keeps its centered geometry and separate body padding.
- **From 761px through 1100px:** page edges (27px); location/footer note hide; hero width (480px); generic panel width (64vw), right (26px). Replay edges reduce to (24px), with a minimum Coach column of (260px) and a (16px) gap.
- **At and below 760px:** header height (76px), side padding (18px), icon-only map-entry action. Hero sits left (25px), top (20%); flight copy top (24%). Camera controls use left (22px), right (20px), bottom (145px). All product panels use left/right (10px), top (86px), bottom (126px), and mobile body padding. Replay video and analysis stack in ordinary panel flow; the video is not sticky. Coach has no overall height cap, while its message log can scroll up to (330px). Screenshots have no display-height cap. Capability lists become one column; plans stack with two-column feature lists. The five-destination dock spans the footer. Dialog width is viewport minus (22px).
- **At and below 390px:** header edges (14px); product panels retain the mobile geometry and padding. Dock icons (16px) stack above labels. Camera controls use bottom (146px), with buttons at least (42px) square and narrower instruction wrapping.
- **At mobile heights at or below 740px:** tighter text/action gaps remain; the final exploration stylesheet sets hero top (20%).

The stage owns pointer and touch gestures with `touch-action: none`. Wheel input over the stage zooms; panel bodies retain native vertical scrolling with `touch-action: pan-y`. Buttons use touch manipulation. Control dimensions are component-specific.

## Elevation & Depth

Depth comes from the native textured map, perspective, warm directional light, cool fill, fog and manually refreshed soft shadows. Translucent tonal layers and selective shadows separate the interface. The vignette darkens text regions without intercepting input.

### Shadow Vocabulary

- **Beacon label:** `0 8px 24px #0003`.
- **Product panel:** `0 18px 65px #0004`.
- **Screenshot dialog:** `0 22px 80px #0009`, with a dark native-dialog backdrop.
- **Original replay marker glow:** player `0 0 20px #ef444480, inset 0 0 15px #ef444433`; teammate `0 0 20px #eab30880, inset 0 0 15px #eab30833`. The paired outer/inner glow belongs to the restored annotation rings.

The canvas uses field of view (55) in landscape and (80) in portrait, initial device pixel ratio bounded to (1–1.5), exposure (0.95) and a cached static directional shadow map (2048px), refreshed on readiness and LOD changes. Sustained active rendering below (30 FPS) can lower moving pixel ratio to (1), then (0.8); idle time and isolated interactions do not trigger this reduction. After motion or playback settles, the initial sharp pixel ratio returns for the final frame before demand rendering sleeps. The measured moving ratio is reused on the next interaction. These settings are not performance guarantees.

### Motion

Button-led flight follows paired camera/attention curves over (3.4s), with cubic ease-in/out and a smooth join from the live camera during the first (24%) of travel. Free camera position, look target and lens zoom converge with `1 - exp(-delta * 3.5)` and animation delta capped at (0.05s) to exclude long idle pauses. Pointer parallax applies outside selected panels and active drag, and ignores touch. Entry copy appears over (0.65s ease-out); flight and inside copy use (0.7s ease-out) and (12px) vertical travel. Tool panels now enter over (360ms), `cubic-bezier(.16, 1, .3, 1)`, from opacity (0.3), perspective (1200px), translation (22px, 0, -22px) and rotation (-2deg). Outgoing tool panels leave over (120ms ease-in), ending at opacity (0), translation (-12px, 0, -16px) and rotation (1deg). Their transform origin is (85%, 50%). Primary actions lift (2px), callouts (3px).

The 3D scene renders on demand: travel actions, drag, wheel/pinch, keyboard camera input, pointer movement, selection, resizing, Steam departure and illustrative utility playback wake it. Flight, camera convergence, tool-camera motion, Steam departure and active playback request subsequent frames, then restore sharp resolution and stop. Beacon breathing and slow dust rotation update only during those active frames and do not sustain a render loop. Playback excludes the initial idle interval when started or resumed.

Tool destination changes start the camera immediately while the outgoing content finishes its (120ms) exit; the incoming content then starts its (360ms) entrance. The camera follows a (560ms) cubic-out arc from its live pose to the existing authored tool endpoint, with a sideways arc of up to (0.85) scene units and vertical lift of (0.35). Zoom and the utility projection offset interpolate along the same eased progress; utility endpoints and final view offset stay as authored. The latest request cancels stale panel timers; Escape or ordinary outside closure cancels a pending switch. Selecting the current tool preserves its mounted replay. Reduced motion makes tool panels and their camera changes immediate. These short authored arcs reuse the existing Fiber camera, memoized vectors, useFrame and demand loop.

Steam sign-in has a separate explicit departure: (1100ms), cubic-out, from the live pose to camera `[-1, 17, 31]`, looking at the CT position plus (0.75) vertical units, or `[4.5, 0.35, 25]`, with lens zoom (1.8). The arc adds up to (-1.2) X and (+1.4) Y scene units; projection offset returns to zero. Header, overlays, footer and camera controls fade over (180ms); 3D destination beacons hide during entry. A World Teal veil rises from opacity (0) through (0.08) at (35%) and (0.86) at (75%) to (1), while the centered StratAI mark appears between (30%) and (60%). At the timed handoff, navigation uses the existing Steam URL once. If 3D is unavailable, normal flight is active or reduced motion is selected, the same handoff uses only (180ms) opacity without camera motion.

Both agents are static: existing clips are sampled near ordinary end poses and paused. CT uses `ct_loadout_pistol01_walkup` at (0.995); T uses `Animacion` at (0.99). Both normalize to height (1.38) scene units. CT stands on A at `[4.5, -0.4, 25]`; T stands on stairs at `[4.15, 2.4, 14.56]`. There is no moving replay agent or replay path in the current presentation.

Reduced motion follows the OS preference and the footer toggle. It removes decorative CSS transitions/animation, parallax, beacon breathing and dust rotation, with immediate travel, tool-panel changes and camera convergence. The Steam connection overlay deliberately retains only its (180ms) opacity fade; its camera remains still. The same bounded controls and destinations remain available. Explicit illustrative utility playback remains user-controlled over (5s). Replay keeps its video checkpoints, reading delays and transport; commentary appears immediately instead of typing. Muted inline autoplay is attempted on opening, with a manual play action when the browser rejects it. The original replay ring pulse runs for (1.5s ease-out), scaling from (1) to (1.5) while fading from opacity (1) to (0); reduced motion disables that pulse and retains the visible ring and label.

**The User-Controlled Motion Rule.** Keep every destination available with reduced motion and preserve explicit playback controls for media and illustrative utility.

**The Cancellable Handoff Rule.** Keep Steam entry cancellable before handoff, preserve the native sign-in href and modifier-click behavior, and keep reduced-motion tool changes immediate.

## Shapes

Controls and reading surfaces have square silhouettes and fine borders. Rounded forms belong to the motion track, state dots, circular beacons and the original replay marker labels. The restored replay rings remain circular; their compact label rounding uses `replay-marker-label`. The wordmark pixel stays square. Screenshots preserve original proportions with `object-fit: contain`; video uses a (16:9) containing area. Interface shape rules do not restyle native map geometry, UVs or textures.

## Components

### Entry, dock and camera controls

The mint “Estoy listo” action has minimum height (52px), icon gap (27px), and hover lift; mobile uses (47px) and (14px). Its neighboring Steam sign-in link retains the original landing label and existing authentication endpoint. Normal primary same-tab activation starts the authored departure before navigating once; modifier, middle-button and new-tab behavior remains native. Header actions open Cómo funciona, Planes y FAQ, and enter A or return to the aerial view. Scene-dependent controls wait for readiness; the header travel action is disabled in flight.

The dock exposes Replay, Mapa, Stats, Coach and Retos as labeled icon buttons with `aria-pressed`. Desktop buttons have minimum height (49px), separators and mint tonal selection. The compact camera bar exposes zoom in/out, reset and “Volver al aire”; buttons are at least (44px) square except narrow layouts (42px). Camera controls and instructions appear only during exploration without an open panel.

Drag pans; right-drag or Shift-drag rotates; wheel and pinch zoom. The focusable stage supports arrow-key pan, Shift-arrow rotation, +/− zoom and R reset. Arrival focuses the stage when no panel is open. Manual view values remain saved while a panel is open; stage gestures are then disabled so the panel keeps native scrolling. Reset restores the authored A view; returning to A starts from that view.

### Steam departure and handoff

The centered connection surface uses the existing World Teal, Mineral White and Journey Mint palette, the larger Inter wordmark, a square brand pixel and the status “Conectando con Steam”. While active, the underlying UI and stage are inert and hidden from assistive technology; the named connection section receives focus. Escape before handoff cancels and restores the trigger focus. A persisted pageshow clears the transition after browser Back, and unmount clears its timer. Repeated activation cannot queue additional navigation. If the browser prevents automatic navigation, “Continuar a Steam” retains an ordinary link to the same endpoint and “Volver a Mirage” cancels the departure. This presentation uses no new renderer, asset or application dependency.

### World callouts

Each destination combines a colored stem, ring, sphere, leader and HTML button. Labels are (170px) wide and at least (46px) high; mobile caps width at (166px) and uses minimum height (44px). Hover preserves the centering transform while lifting the label. Callouts become interactive after arrival; opening a panel hides their layer. Steam departure also disables the HTML callouts and hides the 3D beacons.

### Product information panel

A labeled nonmodal `aside` focuses its close button when opened. Close-button, Escape and outside-pointer closure restore the captured connected trigger, falling back to the first dock button. A capture-phase document `pointerdown` listener closes on ordinary outside input and is removed on cleanup. Direct destination controls in the dock, header and world pins are excluded so one click starts their coordinated switch. Outgoing panels are inert, hidden from assistive technology and pointer-disabled; Steam departure suspends panel dismissal. Interior controls, latest-request switching and same-tool replay preservation remain available; actual replay teardown still pauses media and cancels pending timers. The close control has a name and minimum size (38px). Chrome/footer stay fixed while media and original content scroll.

Information navigation offers Cómo funciona, Planes and Preguntas frecuentes through pressed-state buttons. Guide steps reuse landing copy. Three proposed tiers use a monthly/yearly switch; yearly display follows the implemented ten-month calculation. Plans explicitly activate no subscription or charge. FAQ uses native `details`/`summary`.

### Original screenshots and dialog

Every product destination reuses its original screenshot, alternative text and “Ampliar captura original” action. The native modal `dialog` contains that image, title and focused close button. Browser semantics provide modal behavior and Escape closure; Escape does not close the underlying product panel. Images are neither edited nor generated.

### Tactical video and illustrative utility

Replay embeds original `ChatIA.mp4` with muted inline autoplay, automatic preload and the original replay poster. The transport offers pause/play, early continuation during analysis and restart; chapter actions select the context at (1.3s) or decision at (6s). Autoplay rejection leaves a manual play action; a load failure offers retry. The clip is identified as Dust2 inside the Mirage presentation.

When the first media-time update reaches a checkpoint, analysis pauses the video and seeks back to exactly (1.3s) or (6s). Direct chapter selection delegates to that same operation. This freezes the source frame used for annotation calibration instead of retaining a later timeupdate frame. Original Spanish insights from `replayDemoData.js` appear sequentially, typing two characters every (25ms), or (80 characters/s). The first insight begins after (500ms), and the next after (1500ms); their original reading windows are (6000ms) for context and (8000ms) for the decision, followed by (3000ms) before automatic continuation. A chapter selected again refreshes its timers. Restart cancels pending work and begins a fresh sequence; closure cancels pending work and pauses media. Insight IDs prevent duplicate comments.

The video annotations retain the original ring/label treatment in the current [LandingDemo.jsx](LandingDemo.jsx) and [productPresentation.css](productPresentation.css). The retired `styles/Landing/sections/chatDemo.css` and `components/Landing/sections/chat-demo/ChatDemoSection.jsx` are historical sources retained in the [pre-migration snapshot](../../../../output/landing-official-20261016/before-migration.zip). A (60px) anchor centers each (3px) circular ring on its calibrated blue-ball position; the label uses a bottom offset of (-28px) from that anchor. Blue-pixel centroid measurements in the native (1920 x 1080) clip determine the fixed centers: at (1.3s), corta uses pixel (969, 652), or left (50.46875%), top (60.37037%), and B uses (323, 480), or (16.82292%, 44.44444%). At (6s), corta uses (968, 625), or (50.41667%, 57.87037%), and B uses (301, 641), or (15.67708%, 59.35185%). The critical corta rule is scoped to the critical replay phase. At and below (760px), the B and critical-B labels begin at the anchor's left edge without a horizontal transform so the full label remains within the video; desktop labels stay centered. The [source measurements](../../../../output/playwright/landing-world-replay-20261016/alignment/source-ball-measurements.json) retain the blue-component centroids; the CSS uses the listed integer-pixel anchors derived from those measurements. The player ring and label stay red; the teammate variants stay yellow. The decorative marker ignores pointer input, is hidden from assistive technology and follows the active scripted insight. These are fixed annotations for the two scripted pause frames; the calibration does not establish continuous gameplay tracking or player identity. The pulse is specified by CSS; static captures do not demonstrate its animation. It preserves the original video frame, media, reading delays and commentary; exact checkpoint seeking replaces the earlier late pause frames.

At the end, the original summary and three prepared follow-up questions remain available. Each question appends its prepared response after (1s) and can be asked once per run. Status and message-log semantics accompany the custom controls. The disclosure identifies the comments as a scripted illustration of the Coach proposal, in development; this is presentation playback, not real-time coaching or new factual analysis.

Mapa has a separate play/pause action for illustrative arc/smoke, without physics or parsed-match claims. Stats links to replay, Coach to the tactical demo, and Retos to the guide. Coach and Retos retain development/proposal labels.

### Loading, fallback and accessibility

Loading uses a dark full-screen status and thin progress line, with an ordinary “Conectar Steam” link through the existing sign-in handler and “Ver funciones mientras carga” access to the product content before 3D readiness. The underlying stage and interface remain inert and hidden from assistive technology until ready or explicit feature access; loading controls become inert during Steam handoff. A load error adds retry. Scene failure offers the existing Mirage radar, reload and feature-access actions. A skip link leads to the dock; focus uses Journey Mint (2px), offset (5px), offset (2px) on native FAQ summaries, and inset offset (-4px) on the stage. Status regions announce flight and journey state; the stage has a named keyboard instruction. These mechanisms do not certify WCAG conformance.

### Asset truth and review scope

The map is a local CS2 export from `de_mirage.vpk` through Source 2 Viewer (20.1). The scene loads `mirage-exploration.glb` (71,413,376 bytes), prepared from the retained uncompressed native export. `site_high` preserves all (277,445) original A triangles, face winding and position/normal/tangent/UV attributes through lossless Meshopt compression. The shared-texture `site_distant` alternative has (97,467) triangles; the background has (376,044). Full A switches in below (68) scene units and stays until (78), retaining native detail throughout close exploration. Only distant attributes reduce precision. Foliage wind/variation `COLOR_0` attributes are removed to prevent false red/black PBR tint; original image content, alpha masks and materials remain. The (500) shared KTX2 images have mipmaps, with near texture cap (1024px) and distant-exclusive cap (256px). Web lighting does not reproduce every CS2 shader, blend or lightmap.

The earlier `mirage-lowpc.glb` (31,839,476 bytes) and its [PERFORMANCE.md](PERFORMANCE.md) describe the rejected globally simplified pass. The prior `mirage-textured.glb` (25,506,088 bytes) remains the explicit baseline benchmark. [EXPLORATION.md](EXPLORATION.md) and the current [PROVENANCE.json](../../../../output/landing-world-exploration-20261015/PROVENANCE.json) supersede that optimization pass for current behavior and budgets. [ATTRIBUTION.md](../../../public/landing-world/ATTRIBUTION.md) records resource origin and transformation history; extraction [PROVENANCE.json](../../../../output/landing-world-assets-20261015/PROVENANCE.json) and [OPTIMIZATION.json](../../../../output/landing-world-optimization-20261015/OPTIMIZATION.json) retain the earlier stages.

CT/T models, screenshots and video remain unmodified. Original Spanish/English copy now lives in [copy.es.json](copy.es.json) and [copy.en.json](copy.en.json); the unchanged scripted demo lives in [replayDemoData.js](replayDemoData.js). The retired `components/Landing` and `styles/Landing` trees were removed after the bounded [pre-migration snapshot](../../../../output/landing-official-20261016/before-migration.zip). Their paths describe provenance, not current imports. The current entry headline and camera instructions are presentation copy in `LandingWorld.jsx`. No images were generated. Presentation resources are not Coach analysis inputs. Meshopt is supplied by existing Drei; the current map does not use Draco compression. Existing Draco agent decoding and KTX decoding use at most two workers each; the KTX worker pool is owned above Suspense and disposed on unmount. The local Basis Embind glue uses closures with unchanged WASM under the existing CSP, without adding `unsafe-eval`; comparison against the stock codec covers all (500) images and (14,163) mip/BC1/BC3/BC7 combinations, with (359,024,480) decoded bytes identical. No application dependency or CSP change is required.

The retained current A resource snapshot estimates full-scene texture storage with mipmaps at (190.53 MB), compared with the baseline (558.10 MB), and geometry buffers at (69.12 MB), compared with (97.54 MB). Its (632,820) drawn triangles and (421) calls are local counters, not an FPS result. Restored native A and larger near textures increase the map download to (71.41 MB), under the retained preview-stage budget (75 MB). Texture storage is a format/buffer estimate, not driver-reported total VRAM. Slow-network, compression-fallback and real low-end/mobile hardware validation remain before production; no faster-load or universal-FPS claim is made.

The combined camera envelope also has a retained source-geometry framing check: (2,916) views with (209) opaque double-sided rays per view. Maximum misses are (7) at zoom (0.8), (2) at (1.4) and (0) at (2.1). These samples cover the central 90% horizontal / 80% vertical region at (2560 × 1221); small source gaps remain at wide zoom. This ray proxy complements roof clearance and does not establish a rendered visual verdict.

The retained [exploration detector report](../../../../output/landing-world-exploration-20261015/detector.json) records the prior (2) color and (2) type advisories. The [replay refinement report](../../../../output/landing-world-replay-20261016/detector.json) retains its historical (38) advisories and pinned Inter warning. The [marker restoration report](../../../../output/landing-world-replay-20261016/player-marker-detector.json) records one pre-documentation run with (62) advisories: (38) type, (23) color and (1) radius, plus the same pinned Inter warning. The red/yellow annotation palette, bold label type and label rounding restore the user-pinned original treatment. Other existing advisories remain outside this focused restoration. Detector rules and the hook remain enabled; no suppression or threshold change was added, and no rerun replaces the recorded findings.

The full frontend suite passed (65 files / 402 tests) before the finite camera/layout correction; the following focused suite passed (10 files / 46 tests). For the preceding replay refinement, the [focused suite](../../../../output/landing-world-replay-20261016/focused-final.log) passed (11 files / 54 tests), and the [root](../../../../output/landing-world-replay-20261016/root-build.log) and [isolated preview](../../../../output/landing-world-replay-20261016/preview-build.log) builds passed. That historical preview entry is (853.10 kB), gzip (232.37 kB); its existing (800 kB) chunk advisory is retained. The marker restoration and outside-pointer regression [focused suite](../../../../output/landing-world-replay-20261016/player-marker-tests.log) passed (11 files / 55 tests). The preceding marker-restoration [isolated preview build](../../../../output/landing-world-replay-20261016/player-marker-validation.json) passed with entry (853.39 kB), gzip (232.43 kB), and CSS (41.26 kB); the existing (800 kB) chunk advisory remains. The root build above remains historical; no root-build rerun is claimed for the marker restoration. The preceding alignment [focused suite](../../../../output/landing-world-replay-20261016/alignment-tests.log) passed (11 files / 55 tests), with exact checkpoint assertions extended. The preceding alignment [isolated preview build](../../../../output/landing-world-replay-20261016/alignment-final-build.log) passed with entry (853.40 kB), gzip (232.43 kB), CSS (41.49 kB) and the unchanged (800 kB) chunk advisory. The root build was not rerun for that alignment pass. [Historical Chrome playback evidence](../../../../output/playwright/landing-world-replay-20261016/browser-playback-evidence.json) records pauses at (1.485868s) and (6.185971s), unchanged media time while commentary is shown, automatic continuation, clip end at (10.9s), and no recorded browser errors. Those late pause times are historical and superseded by the exact checkpoint seek contract; the retained run also predates the final height-budget adjustment. Current browser alignment evidence and its review remain separate from these source-derived notes. Current validation, runtime captures and the finish verdict remain in the separate review evidence; this documentation does not assert a final visual verdict or source-hash match. That preview-stage evidence remains historical and does not establish the official migration's final verification.

The preceding standalone motion [focused suite](../../../../output/landing-world-motion-20261016/focused-final.log) passed (13 files / 68 tests). The preceding motion [root build](../../../../output/landing-world-motion-20261016/root-build.log) and [isolated preview build](../../../../output/landing-world-motion-20261016/preview-build-final.log) passed; the preview entry is (857.34 kB), gzip (233.51 kB), with CSS (43.83 kB) and the unchanged (800 kB) chunk advisory. The [single motion detector run](../../../../output/landing-world-motion-20261016/detector.json) retains (52) advisories: (39) type and (13) color, plus the existing pinned Inter warning. The new entry typography and authored motion are intentional scoped additions; no suppression or threshold change was added. The retained [browser evidence](../../../../output/landing-world-motion-20261016/browser-evidence.json) records normal/reduced handoff requests to the existing auth URL, cancellation and Back reset, with no recorded errors. Its auth route is intercepted and fulfilled by a local test response; it verifies the destination and return behavior, not a completed production Steam login. Final screenshots, source hashes and the independent visual verdict remain in [REVIEW.md](REVIEW.md).

The official migration mounts this feature at `/` through `App.jsx` and its lazy `index.js` entry. The public root still performs no session check; the existing Steam callback continues through `SteamLoginSuccess` to history-code onboarding or dashboard, and protected routes retain `RequireAuth`. No backend session, acquisition or complete demo-pipeline contract changes are introduced. The [official migration evidence directory](../../../../output/landing-official-20261016/) and [REVIEW.md](REVIEW.md) own final post-loading tests/builds, API/browser verification and the independent review. This source-derived documentation does not declare pending results, a remote launch, external Steam authentication or low-end hardware validation.

## Do's and Don'ts

### Do:

- **Do** apply this reference to the official root landing feature and keep its styling scoped to that feature.
- **Do** preserve the five destination colors together with labels, icons and explicit selection states.
- **Do** keep the bottom feature controls, keyboard focus path and reduced-motion destinations available.
- **Do** retain original Spanish copy and media with the Coach development, retos proposal and scripted-demo disclosures.
- **Do** preserve the recorded native-map provenance and original media hashes.
- **Do** preserve the original red/yellow marker styling and the calibrated coordinates at the exact scripted pause frames.
- **Do** keep Steam entry cancellable before handoff and preserve native links and reduced-motion alternatives.

### Don't:

- **Don't** extend landing-specific styles to other application routes or treat this reference as AI Coach delivery-status evidence.
- **Don't** present screenshots or scripted comments as demonstrated automatic coaching.
- **Don't** imply that illustrative utility validates gameplay physics.
- **Don't** require pointer parallax or ambient animation to understand or operate the landing.
- **Don't** replace native map depth with a generated background while claiming the implemented 3D behavior.
- **Don't** infer production readiness from local rendering or the compressed GLB size.

