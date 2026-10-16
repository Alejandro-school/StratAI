# Review — refined Mirage landing preview

## Official root migration — 2026-10-16

Fresh independent disposition: **ship**, limited to replacing the former root
landing with the approved Mirage experience. All five review sections were
returned; no material fixes or recaptures were required. The user clarified
that production meant the official application landing, not remote deployment.

`App.jsx` now lazy-loads the feature entry at `/`. A comparison against the
retained pre-change source confirms that only this import changed in App:
existing routes and providers remain intact. Original Spanish/English copy
and the replay script moved byte-identically into the feature before removing
24 legacy-only component/style files. Native media and Mirage assets remain.
The optional standalone entry is secondary to the official application entry.

Styles, root variables and reduced-motion rules are scoped to `.landing-world`;
the feature no longer locks document scrolling or restyles account controls.
The official HTML loads Inter/Rajdhani. A fixed 10.4 px annotation label retains
the prior preview size under the application's smaller mobile root font.
The prototype footer label is replaced. Loading offers the existing Steam
access and product panels before the map finishes; obscured controls are inert.

Final verification passes: 69 frontend files / 435 tests, 37 selected backend
OpenID/session tests, the main frontend build, and maintained documentation
links. Main entry is 189.46 kB (63.24 kB gzip), deferred landing 39.16 kB
(12.74 kB gzip), scene 142.55 kB and shared Three 659.69 kB. The existing
800 kB advisory threshold is unchanged. One detector run records 53
advisories and the existing pinned-Inter warning, without suppression.
It predates only the final narrow-loading note centering; it was not rerun.

Chrome 155 verified the official local `/` against real FastAPI and Redis:
public entry makes no session request; API readiness/status succeed; protected
pipeline status denies unauthenticated access. The actual login endpoint
returns 307 to Steam after 1139 ms with callback state and an HttpOnly,
SameSite=Lax session cookie. CORS permits the configured frontend. A malformed
callback returns 400 without authentication. Only the external provider page
was intercepted after the real backend redirect with an explicit test receipt;
no Steam account was authenticated. Successful history/dashboard destinations
are covered by explicit session fixtures in the route integration tests.

The real unauthenticated callback recovery renders identically after SPA
navigation from the landing and on a fresh direct visit: canvas removed,
account control styles equal and document scrolling available. Two expected
StrictMode diagnostics report the deliberately missing session; there are no
unexpected page errors. The official root and real API readiness return 200
in the final health check. Standard full startup is used; existing queued
work/startup errors do not establish a new demo-analysis functional closure.

Main thread and fresh reviewer opened all eight final JPEGs in
`.impeccable/review/official-20261016/`: aerial, exact replay pause with full
original comment, live Steam departure and deliberate missing-session recovery
at 1440×900; settled Map at 2560×1221; aerial and scrolling Plans at 390×844;
slow-scene loading access at 320×740. The departure is a live intermediate
alpha state. All 31 recorded source/document/capture hashes matched during
review. No remote deployment, completed external authentication, physical
mobile/low-end GPU performance or whole-backend launch certification is claimed.

Evidence and rollback inventory: `output/landing-official-20261016/VALIDATION.json`,
`browser-evidence.json`, `final-health.json`, `retired-files.json` and
`before-migration.zip`. Scoped design docs and the frontend guide describe the
official route, style ownership, moved content and environment requirements.

## Approved camera and panel motion — 2026-10-16

Fresh independent disposition: **ship**, limited to the requested motion
extension in the isolated preview. All five review sections were returned;
no material fixes or recapture requirements were identified within that scope.

Normal Steam access starts a 1100 ms approach from the current camera toward
A/CT, hides the interface and tool beacons, darkens the scene and presents
the connection mark before following the existing authentication href.
Reduced motion uses a 180 ms opacity transition with a stationary camera.
Native modified links, cancellation, restored focus, single handoff and
persisted browser-back recovery are retained. Tool changes coordinate
120 ms panel exit, 360 ms entry and a 560 ms camera arc, retaining the
approved endpoints and original content. Rendering remains finite and
uses the existing demand Canvas.

All 13 focused files / 68 tests passed. Root and final isolated builds passed;
the independent entry is 857.34 kB (233.51 kB gzip), scene chunk 275.33 kB
and CSS 43.83 kB. The unchanged 800 kB advisory remains disclosed. One
detector run recorded 52 advisories and the existing Inter warning without
suppression. Scoped DESIGN.md, design.json and README.md describe the
source-derived behavior and evidence limits.

The main thread and fresh reviewer opened all 12 final JPEGs in
`.impeccable/review/motion-20261016/`: settled Replay, Map, Coach and Plans
at 1440×900; Plans and live departure at 390×844; three live desktop
departure stages; settled Map and departure at 2560×1221; departure at
320×740. Start/mid/late images intentionally capture live choreography;
intermediate brand opacity is not a settled reading state. First-round
and failed-harness outputs are retained separately and excluded.

Chrome 155 evidence records moving camera samples, normal handoff at
1128 ms, reduced handoff at 194 ms with identical camera coordinates,
tool switching, media teardown, outside dismissal, cancellation and
browser back, with no recorded browser errors. Authentication responses
were intercepted by clearly labelled local test pages: no real Steam
session was authenticated. Native mobile hardware, low-end GPU performance
and whole-app or production readiness are not established.

Evidence, exact source/documentation/capture hashes and scoped verdict:
`output/landing-world-motion-20261016/VALIDATION.json`,
`browser-evidence.json`, `size-evidence.json` and `finish-review.md`.
The original replay clock, calibrated annotations, assets and product
copy remain intact; the production root and backend contracts are unchanged.

## Ball alignment corrected — 2026-10-16

Fresh independent disposition: **ship**, limited to centering the four scripted
player annotations on the actual blue balls. This supersedes the previous
marker-restoration review's assumption that inherited coordinates were adequate.
The user's screenshot correctly showed both rings displaced left/up.

The native 1920×1080 video supplies calibrated centers for own player and
teammate at each pause. Analysis now seeks exactly to 1.3s or 6s after pausing;
late timeupdate frames no longer shift the balls under fixed annotations.
Critical corta has its own position. Original ring colors, 60px size, pulse,
labels, commentary timing, automatic continuation and outside dismissal remain.
The mobile teammate label stays inside the video width.

All 11 focused files / 55 tests passed, including exact-pause regression
assertions; the isolated preview build passed. A complete natural Chrome run
records both exact pauses and automatic resumes, end at 10.9s, reduced motion,
outside dismissal and no browser errors. Independent blue-pixel measurements
place overlay centers within 0.603 native pixels / 0.261 CSS pixels of the
balls. This verifies these four scripted annotations, not gameplay tracking.

All eight final files in `.impeccable/review/ball-alignment-20261016/` were
opened by the main thread and fresh reviewer: own/teammate at 2560×1221,
both critical annotations at 1440×900, mobile own at 390×844, mobile teammate
at 320×740, and two direct video-region captures. The reviewer compared actual
balls with the rings and returned no material fixes or recapture requirements.
Frozen captures do not demonstrate the moving pulse. No whole-app or
production readiness is claimed.

Current source hashes, asset hash, calibration, logs and scoped verdict:
`output/landing-world-replay-20261016/alignment-validation.json`.
Native source measurements and complete browser proof:
`output/playwright/landing-world-replay-20261016/alignment-evidence.json`
and its `alignment/` directory. First-round captures remain retained there;
the final batch additionally verifies horizontal label containment. One detector
run recorded 51 advisories and the existing Inter warning, without suppression.
Scoped design docs describe the calibrated positions and exact-frame contract.

## Original player marker restored — 2026-10-16

The user rejected the earlier replacement of the original red circle by a mint
label. A fresh independent review returns **ship**, limited to restoring that
original ring treatment and preserving the existing outside dismissal.
No material fixes or recaptures remain in this scope.

The original 60px red/yellow circle, 3px border, palette/glow, label, fixed
positions and 1.5s pulse are restored from `chatDemo.css:309–402`. Only the
overlay markup and its scoped styles changed. The preview build and all
11 files / 55 tests passed; recorded browser errors are empty. The reviewer
opened all five final screenshots and both rejected green-label comparisons:
`.impeccable/review/player-marker-20261016/`. Captures establish visual
restoration, not gameplay identity/tracking; fixed coordinates are inherited
from the original. CSS computation verifies pulse configuration and its
reduced-motion disablement; frozen screenshots do not show moving animation.

Current checks, source hashes and build metrics are in
`output/landing-world-replay-20261016/player-marker-validation.json` and
`output/playwright/landing-world-replay-20261016/player-marker-evidence.json`.
One detector run recorded 62 advisories and the existing Inter warning;
the user-pinned original marker treatment is retained without suppression.
This is a local preview correction, not a new whole-surface or production verdict.

## Outside dismissal — 2026-10-16

Functional follow-up after the visual review below: pressing outside any product
panel closes it through the existing close path. Interior controls, direct tool
switching, focus restoration and replay teardown remain available. All 11
focused files / 55 tests and the independent preview build passed. Chrome checks
passed with mouse at 1440×900 and touch at 390×844, with no browser errors:
`output/playwright/landing-world-replay-20261016/outside-dismiss-evidence.json`.
The review and source hashes below describe the preceding visual refinement;
they are not a new independent review of this event-handler change.

## Replay and product presentation — 2026-10-16

Fresh independent finish disposition: **ship**, limited to the requested local
preview refinements. No material findings, required fix batch or recapture.
This is not production-launch approval.

Product panels and original screenshots are larger. Replay uses a wide desktop
workspace with the original Dust2 video and Coach together. The muted video
starts on opening; its actual media clock triggers the original 1.3s and 6s
analysis pauses, progressive comments and automatic continuation. Original
prepared questions remain after the round. Manual pause, restart, chapter
selection, autoplay fallback and cleanup are covered. Mobile uses ordinary
vertical scrolling so the video does not obscure the analysis. The original
static replay screenshot and capabilities remain in a native disclosure.
The global header's bottom border has been removed.

Verification: 11 focused frontend test files / 54 tests passed; root and
independent frontend builds passed. The independent entry is 853.10kB
(232.37kB gzip), against the unchanged 800kB advisory. Natural playback in
Chrome 155 paused at 1.485868s and 6.185971s, remained stopped during typing,
resumed automatically and ended at 10.9s. Original follow-up conversation
worked; closing removed the video. No browser errors were recorded.

All nine final JPEGs in `.impeccable/review/replay-20261016/` were opened and
validated by both the implementation thread and fresh reviewer: replay,
critical moment, conversation and map at 2560×1221; replay and map at
1440×900; replay and scrolled Coach at 390×844; replay at 320×740. The
1440 map image continues below its native scrolling viewport. Final captures
follow the layout corrections; complete natural playback evidence predates
only those CSS/presentation changes, with LandingDemo.jsx unchanged.

Current hashes and logs: `output/landing-world-replay-20261016/VALIDATION.json`.
Browser evidence: `output/playwright/landing-world-replay-20261016/` contains
`browser-playback-evidence.json` for natural playback, `browser-evidence.json`
for final responsive states, and the capture harness. Failed harness and
first-round captures remain retained and are excluded from final evidence.
One detector run returned 38 advisories and the pre-existing Inter warning;
the user's Inter/Rajdhani identity is retained without suppression.

The scoped DESIGN.md, design.json and README describe the new layout and
restored scripted demonstration. Mirage geometry/textures, agent models,
original video/screenshots and factual product copy remain unchanged. No
dependencies, CSP exceptions, root-route replacement, deployment, backend
producer or canonical AI Coach contract/status changed. No new raster was
generated. GPU performance, whole-suite and production readiness are not
established by this focused review.


## Earlier: button-led A exploration

Independent finite verdict: **ship**, limited to the two material corrections
in this local preview. The reviewer scored both resolved; this is not a new
whole-surface or production certification. The earlier optimization below is
historical and was superseded after the user's fidelity rejection.

The preview now opens with aerial marketing and an explicit “Estoy listo”
flight into A. It hands over to damped pan, rotation and zoom, bounded by a
coupled asymmetric envelope around the approved A view. Closing information
restores the previous camera. Panels retain original product media and native
scrolling; moving DPR restores the sharp still frame before rendering sleeps.

The resource restores all 277,445 original A faces and retained attributes,
verified independently after lossless Meshopt encoding. Background and the
distant-A alternative stay reduced, sharing 500 native KTX2 images. Download
is 71,413,376 bytes versus 31.8MB previously; estimated scene textures are
190.53MB versus 558.10MB baseline and geometry 69.12MB versus 97.54MB.
These storage estimates do not establish real low-end FPS or faster loading.

The fresh full review returned `fix` for exposed outer-map geometry during
close exploration and callouts crowding camera guidance. One bounded batch:

1. Couples focus, yaw, pitch and spare dolly so their individual extremes
   cannot combine into the rejected outer-shell view. Lens zoom remains 2.1×.
2. Reserves the full guidance/control area, including label width and a gap;
   adapts short mobile rows while keeping both agents visible.

The same reviewer opened all seven replacement captures and scored **P1
resolved / P2 resolved**. Close/max views retain interior A framing without
the previous large void. Desktop labels clear controls; 390px and 320px mobile
views preserve separation and the CT. It confirmed preservation of native
crate/palm detail, branding, agent positions, aerial CTA and product imagery.
No further implementation or recapture was required for these two findings.

Verification: full frontend suite 65 files/402 tests passed before the finite
correction; final focused suite 10 files/46 tests passed after it. Root build
passed; final independent build passed with an 846.63kB entry against the
unchanged 800kB advisory. The repeated source-geometry roof check covers 972
samples with zero below-roof positions. A complementary 2,916-view ray proxy
has maximum misses 7/209 at zoom .8, 2/209 at 1.4, and 0/209 at 2.1. Opaque,
double-sided sampled coverage is not a general collision or raster proof.
The independent reviewer verified all eight recorded runtime hashes.

Browser evidence: button flight, native left-drag/wheel and focused-stage
Shift-arrow rotation, explicit zoom cap/reset, utility start/completion,
native panel scroll (7px available) and exact serialized navigation restoration.
Final mobile documents match 390×844 and 320×740 without overflow; the lowest
320px callout ends at y455 before camera guidance begins at y486.2. Settled A
records zero frames. Physical right-drag/touch are unit tested; real modest
hardware and slow networks remain unverified. No universal FPS claim.

All valid captures are in `.impeccable/review/exploration/`:
`desktop-hero.jpg`, `desktop-world.jpg`, `desktop-close.jpg`, `desktop-max.jpg`,
`desktop-utility.jpg` (2560×1221), `mobile-world.jpg` (390×844) and
`mobile-320.jpg` (320×740). Previous captures are retained in the output's
`before-finite-captures/`. One detector ran before the correction; remaining
fluid-type/color findings were advisory. No second detector or suppression.

Current implementation/provenance: [EXPLORATION.md](EXPLORATION.md),
`output/landing-world-exploration-20261015/PROVENANCE.json`, validation JSON
and final logs. Scoped DESIGN.md, design.json and README were synchronized
and independently checked. Original sources, agents and media stay read-only;
no dependencies/CSP, root route, deployment or AI Coach contract changed.

## Earlier low-PC optimization (historical)

Independent finish verdict: **ship the bounded local preview optimization**.
No required code fixes, recapture or rebuild. This supersedes the resource
description below, which records the earlier unsimplified textured-map pass.

The final map uses 735,372 triangles (−43.1%), native UV/materials and 489 KTX2
images with mipmaps. In the same settled A view, drawn triangles fall from
1,195,098 to 606,709 (−49.2%); estimated full-scene material texture storage
falls from 558.10 to 131.27MB (−76.5%); geometry buffers fall from 97.54 to
57.72MB. Demand rendering records zero frames after the scene settles and
wakes for scroll, pointer interaction, selection, resizing and playback.

The reviewer accepted modest surface softening around the upper stack and
masonry. It confirmed preserved silhouettes, A markings, stairs, boxes, agent
placement, native materials, desktop composition, mobile pins and utility
content/smoke. Tradeoffs remain explicit: 313→339 interior draw calls and
25.51→31.84MB map download. Neither faster initial loading nor better CPU
frame timing is claimed. Real low-end GPU/network validation remains open
before production; local resource measurements do not guarantee every PC.

Final verification: 61 frontend test files, 394 tests passed; root frontend
build and independent preview build passed. The existing 842.67kB entry
warning remains against the unchanged 800kB advisory. New tests protect asset
budgets, texture formats/mips, demand-loop wake/cleanup and active-only DPR
adaptation. One scoped detector run returned no findings. Original and
CSP-compatible Basis glue produced identical bytes for all 489 images in
13,728 mip/BC1/BC3/BC7 comparisons; the unchanged production CSP successfully
loads the compiled preview. No captured browser warnings/errors.

Valid final screenshots are in `.impeccable/review/optimization/`:
`desktop-hero.jpg`, `desktop-world.jpg`, `desktop-utility.jpg` (2554×1218
content), `mobile-world.jpg` (384×831 content under 390×844 emulation).
All were opened and checked by both the implementation and fresh reviewer.
Browser checks covered scroll descent, returning aerial, panel selection,
utility start/completion and smoke, scene idle, and responsive resize.
Temporary viewport and diagnostic query were cleared on the delivered tab.

Full evidence and limitations: [PERFORMANCE.md](PERFORMANCE.md),
`output/landing-world-optimization-20261015/OPTIMIZATION.json`, the paired
renderer JSON/screenshot references and `basis-validation.json`. The VPK,
baseline textured map, original agents and product media remain unchanged.
No dependencies were added to the application; tools remain isolated in
retained output. No deployment, root-route or AI Coach contract change.

## Earlier textured-map refinement

Disposition: **ship**, limited to the independent reviewer's finite verdict on the three material fixes below. This is a local design preview, not production-launch approval or a replacement of the root landing.

## User-requested refinement

- Both existing agents are normalized to a plausible 1.38-unit height after posing. The T stands on the native stairs surface at `[4.15,2.4,14.56]`; the CT stands on A at `[4.5,-.4,25]`. Ordinary sampled clips replace the celebration and moving illustrative agent.
- The overview ends at camera `[-6,17,34]`, target `[3,0,26]`, including A, stairs and CT boxes. It arrives by 82% scroll and holds through the remaining exploration area.
- Five destinations reuse the existing landing's descriptions, capabilities and original screenshots. Replay includes the original Dust2 video and prepared commentary. Guide, plans, yearly/monthly pricing and seven FAQ remain accessible from the header. Native screenshot enlargement, keyboard closing, focus restoration and direct feature navigation remain available.
- Mirage now uses native geometry, UVs and authored textures exported read-only from the local CS2 VPK, compressed to Draco/WebP. The old simplified GLB is retained as reference and is no longer loaded.

## Independent finite verdict

The fresh review returned `fix` for T callout occlusion at the user's wide viewport, utility playback hidden behind the larger panel, and stale scoped design documentation. Its verdict pass scored all three **resolved**:

1. Coach's preferred desktop leader drop is 130px. The T's complete silhouette remains visible at 2560px; desktop/mobile labels stay separate.
2. Utility uses target `[8,2,25]` and a responsive `PerspectiveCamera.setViewOffset` based on panel width and gutter. Its illustrative arc lies in the open site, and the complete smoke volume remains in the unobscured scene. The offset clears when leaving utility.
3. `DESIGN.md` and `.impeccable/design.json` describe the final five destinations, panel geometry, native media controls, static posed agents and textured-map implementation. A scoped documenter confirmed consistency after the last correction.

The reviewer validated all 11 refreshed JPEG captures and observed no regression from that fix batch. This is a finite-fix verdict, not a new whole-surface or production certification.

## Verification

- `npm run test:frontend`: 59 files, 380 tests passed before the finite framing corrections.
- `npm run test:frontend -- src/features/landing-world`: final 4 files, 26 tests passed. Includes aerial-to-overview travel, landmark framing, label separation, agent visibility at 2560/1440/390/320 widths, utility arc/smoke framing at 800/1024/1440/2560 widths, original content coverage and pre-readiness scene-failure recovery.
- `npm run build:frontend`: passed.
- `npx vite build --config landing-world.vite.config.js` from `frontend`: final standalone build passed. Entry is 842.30kB above the existing 800kB advisory; the threshold was not changed.
- One design detector run: no primary violations; 17 color and 21 type advisories reflected intentional additions to the scoped design references. No detector suppression was added.
- Real browser checks: screenshot dialog and Escape, original video readyState 4 and paused seek at 1.3/6 seconds, annual values 0/100/250, guide/FAQ navigation, and Coach → Replay → Escape restoring connected Coach dock focus. No captured console warnings or errors.

## Evidence and limits

Final valid JPEGs are under `.impeccable/review/`: `desktop-hero.jpg`, `desktop-world.jpg`, `desktop-coach.jpg`, `desktop-utility.jpg`, `desktop-challenges.jpg`, `desktop-plans.jpg` (1440×900); `mobile-world.jpg`, `mobile-replay.jpg`, `mobile-plans.jpg` (390×844); `narrow-hero.jpg` (320×740); `user-world.jpg` (natural 2560×1221). Content excludes scrollbar pixels. Scratch PNGs and earlier captures are excluded. Some browser-emulation captures were malformed or stale and were replaced; every final JPEG was inspected independently.

Structured checks: `output/landing-world-review-20261015/refinement/VALIDATION.json`. Asset commands, hashes and counts: `output/landing-world-assets-20261015/PROVENANCE.json`; public attribution: `frontend/public/landing-world/ATTRIBUTION.md`.

The map is 25.5MB with 507 WebP images up to 512px and about 511MB estimated RGBA texture memory with mipmaps. GPU compression/LOD, loading budgets and real-device frame-rate/Core Web Vitals checks remain before production. Web lighting does not reproduce every CS2 shader, blend or lightmap. The Coach and challenges are described as proposals; the replay commentary and utility physics remain illustrative.

No backend producer, demo contract, runtime match data, root route or canonical AI Coach phase status changed. The one-demo factual smoke gate does not apply to this presentation-only work. Nothing was published.
