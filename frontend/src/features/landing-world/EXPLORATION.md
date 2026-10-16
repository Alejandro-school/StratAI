# Button-led A exploration — 2026-10-15

The preview now enters through “Estoy listo”, flies to A in 3.4 seconds and
hands over to a bounded camera. Drag pans; right-drag or Shift-drag rotates;
wheel and pinch zoom. Arrow keys, Shift-arrows, +/− and R are available on the
focused stage, alongside explicit zoom, reset and aerial-return buttons.
Panels keep native scrolling and restore the previous exploration state on
close. Reduced motion offers immediate travel and camera adjustments.

The camera's focus stays within A; yaw, pitch, distance and lens zoom are
bounded. A 17-unit camera-height floor and limited lens zoom avoid Palace/CT
roofs while still allowing a closer view. Pan, orbit, pitch and spare dolly
share an asymmetric ellipsoid around the approved A view, so combining their
individual limits cannot push the framing past Palace's open outer shell.
Lens zoom reaches 2.1×. A real-map
check sampled 972 combinations, including cursor offsets, with zero camera
positions beneath opaque roofs (0.6-unit margin). This is a sampled safety
check, not a general-purpose collision engine.

An additional source-geometry coverage proxy casts 209 rays across each of
2,916 camera/pointer combinations. At zoom 2.1 none miss opaque geometry;
the maximum misses are 2 at 1.4 and 7 at 0.8 (out of 209). This double-sided,
opaque-only sample is supporting evidence, not a rendered-image verdict.
Callouts reserve the complete camera guidance/control region and adapt
their lower rows on short mobile viewports.

## Fidelity and performance

`mirage-exploration.glb` starts from the retained uncompressed native export.
Its `site_high` has all 277,445 original A triangles. Independent validation
checks every face/winding and every position, normal, tangent, UV and remaining
attribute against the prepared source. Meshopt compression is lossless for A;
cyclic index rotations preserve the exact faces. Wind/variation vertex colors
are removed from foliage materials so they no longer tint leaves red/black.
The original texture imagery and alpha masks remain; near textures cap at
1024px, distant-exclusive textures at 256px.

The shared-texture `site_distant` alternative has 97,467 triangles; background
has 376,044. Full A replaces its distant alternative while still small on
screen (68/78-unit hysteresis), and stays full-detail throughout exploration.
The alternatives share textures in one GLB. Distant positions/normals/UVs and
tangents have reduced precision; native A attributes are untouched.

Demand rendering continues only through motion/playback, then restores the
initial sharp DPR and sleeps. The measured lower moving DPR is reused on the
next interaction. KTX2 uses the existing strict-CSP decoder; Meshopt uses the
decoder already supplied by Drei. Application dependencies/CSP are unchanged.

The download is **71,413,376 bytes** versus 31,839,476 for the rejected globally
simplified version. Preserving native A geometry and higher-resolution images
requires this explicit network tradeoff. The new resource budget is 75MB for
this local fidelity preview. Real slow-network and low-end hardware validation
remain before production. No faster initial-load or universal FPS claim.

Evidence and commands: `output/landing-world-exploration-20261015/`.
`geometry-validation.json` verifies original A; `camera-volume-validation.json`
records the 972 camera samples. Basis validation covers 500 images and 14,163
mip/BC1/BC3/BC7 comparisons: 359,024,480 bytes identical to the stock codec.
The original VPK, baseline maps, agent models and product media stay read-only.

Verification: full frontend suite 65 files/402 tests passed before the finite
camera/layout correction; final focused suite 10 files/46 tests passed after
it. Root and isolated builds passed. The standalone entry is 846.63kB;
the existing 800kB advisory is retained.
This preview is not deployed and does not replace the root landing or alter
AI Coach contracts.
