# Mirage landing performance — 2026-10-15

This is a local preview optimization. It preserves the approved camera path,
agent poses, native materials, product copy and media. No route was promoted
and nothing was deployed.

## Measured comparison

Same compiled preview, Brave browser, 2554 × 1221 content viewport, camera
`[-6,17,34]`, target `[3,0,26]`, settled A overview, full-motion setting.
Counters come from the renderer, not an FPS estimate from the machine's name.

| Resource | Previous | Optimized |
|---|---:|---:|
| Map triangles in the asset | 1,292,578 | 735,372 (−43.1%) |
| Triangles drawn in A, including agents | 1,195,098 | 606,709 (−49.2%) |
| Estimated scene texture storage with mips | 558.10 MB | 131.27 MB (−76.5%) |
| Scene geometry buffers | 97.54 MB | 57.72 MB (−40.8%) |
| Draw calls in A | 313 | 339 (+8.3%) |
| Idle frames per 1.5-second diagnostic window | 319 in baseline sample | 0 after settling |
| Map download | 25,506,088 bytes | 31,839,476 bytes (+24.8%) |

Texture storage sums the formats and mip buffers of scene materials; it is
not driver-reported total VRAM and excludes render targets and decoder heaps.
The browser masks the GPU identity. No low-end hardware FPS or universal
compatibility claim follows from these local measurements. The map download
grows because GPU-compressed textures trade network size for a much smaller
working set; this is an explicit tradeoff, not a loading-time improvement.

## Implementation

- Split original geometry into 24-unit spatial tiles, simplify with locked
  boundaries and UV/material seams, then batch pieces below 2,000 triangles
  by material. Large pieces retain frustum culling. A has a stricter error
  bound than distant scenery. The initial unbatched attempt drew 552 calls;
  the final arrangement limits the CPU cost while halving visible triangles.
- Keep native texture detail up to 512px around A; only images exclusive to
  distant materials are capped at 256px. All 489 retained images have full
  KTX2 mip chains: ETC1S for color/emissive, UASTC for normal/ORM maps. Unused
  textures are pruned. No generated or replacement map imagery is used.
- Render on demand. Scroll, pointer movement, selection, resizing and utility
  playback wake the scene. Camera convergence and playback request subsequent
  frames, then stop. Decorative dust/beams advance during those active frames.
- During sustained active rendering below 30 FPS, lower DPR from at most 1.25
  to 1, then .8. Idle time and isolated interactions do not lower resolution.
  Static shadow caching and the accessible non-WebGL fallback remain.
- Limit Draco and KTX workers to two each. Own the KTX pool above Suspense and
  release it on unmount. Preserve production CSP: the vendored Basis Embind
  glue uses closures instead of dynamic JavaScript compilation, with unchanged
  WASM. See `public/basis/LOCAL-CHANGES.md`.

## Evidence and reproduction

`output/landing-world-optimization-20261015/OPTIMIZATION.json` records asset
SHA-256 hashes, settings and retained attempts. The original extraction
provenance remains in `output/landing-world-assets-20261015/PROVENANCE.json`.
Offline tools are isolated there; application dependencies were not changed.

Geometry: `scripts/landing-world/optimize_map.mjs`; texture container conversion:
`compress_map.mjs`; final grouping: `batch_map.mjs`. The retained `etc1s-rdo.log`
and `uastc-rdo.log` identify the encoding stages; settings are in the JSON.
KTX Software 4.4.2, glTF Transform 4.5.0, Three 0.160.1.

For local comparison, append `?perf=1` to `landing-world.html`. The diagnostic
JSON appears on `.world-stage[data-performance]` every 1.5 seconds; it makes no
requests or telemetry calls. `?perf=1&benchmark=baseline` loads the preserved
previous map and continuous render loop. Compare the same view, motion setting
and viewport after loading, with no encoders or tests running concurrently.

- `npm run test:frontend`: 61 files, 394 tests passed.
- `npm run build:frontend` and the standalone Vite build: passed. The existing
  standalone entry warning remains (842.67kB above an unchanged 800kB advisory).
- Asset-budget tests enforce fewer than 60% of the original triangles, at most
  500 primitives, native texture UVs, KTX2/mips, 512px ceiling and <33MB download.
- Demand-loop tests cover wake on play/pause/restart, idle clock exclusion,
  bounded resolution changes, sustained activity and callback cleanup.
- `node scripts/landing-world/verify_basis_csp.mjs`: 489 images, 13,728 mip/format
  comparisons across BC1/BC3/BC7, 210,857,720 bytes identical to the stock codec;
  stock JS fails the strict control, patched JS succeeds with string generation
  disabled. Actual compiled-browser loading also succeeds under production CSP.

Real low-end hardware, unsupported compression-format fallback and slow-network
testing remain production validation work. In browsers with no suitable GPU
texture compression, Three can expand textures and memory savings will differ.
