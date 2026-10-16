# Local CSP-compatible Basis glue

Source: `three@0.160.1/examples/jsm/libs/basis/`, Basis Universal (Apache-2.0).
The WASM binary is unchanged. `basis_transcoder.js` has four generated Embind
factories expressed as ordinary closures instead of runtime `Function`
constructors, plus a static worker global fallback. This corresponds to the
purpose of Emscripten `DYNAMIC_EXECUTION=0`; it is not a custom codec.

Reproduce with `node scripts/landing-world/prepare_basis_csp.mjs`. The script
pins the upstream JS SHA-256 and refuses an unreviewed version change.
`node scripts/landing-world/verify_basis_csp.mjs` first proves that stock glue
fails with string code generation disabled, then compares every image and mip
in the shipped map in BC1, BC3 and BC7 against the stock decoder. It must be
byte-identical. The browser additionally loads it under the production CSP.
No `unsafe-eval` permission was added. Decoder ownership stays above Suspense;
its two-worker pool is disposed when its scene unmounts.

See `LICENSE` (Basis Universal), `THREE-LICENSE` and the upstream README.
