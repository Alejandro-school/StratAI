---
name: stratai-react-three-fiber
description: Change or debug StratAI React Three Fiber scenes, 3D camera interaction and animation. Excludes the Canvas 2D replay renderer and ordinary DOM animation.
metadata:
  author: StratAI
---

# React Three Fiber scenes

Use this for the existing React/Three.js scene integration. Inspect frontend/package.json and the affected scene first; the reviewed stack is React 18, Fiber 8, Drei 9 and Three.js 0.160. Version-check current documentation before adopting a newer API.

## Respect scene ownership

- Extend the existing Canvas and scene components. Use useThree and useFrame inside Canvas; do not add a second WebGLRenderer or requestAnimationFrame loop to a Fiber-owned scene.
- Keep React state for semantic UI changes. Update transient scene values through refs/useFrame, with delta-based animation; avoid setState and fresh vectors/materials in each frame. See [Fiber performance pitfalls](https://r3f.docs.pmnd.rs/advanced/pitfalls).
- Respect the current frame loop mode. For demand rendering, invalidate when imperative changes require a frame. Inspect existing camera control ownership before adding another controller.
- Preserve the project's CS2-to-scene coordinate transforms, units and camera conventions. Check known landmarks instead of guessing axes from generic examples.

## Assets and interaction

Reuse existing loaders and asset caches. Distinguish per-instance animation state from shared geometry/materials. Do not dispose shared cached assets while another mounted consumer uses them; release resources owned by the component at the appropriate lifecycle boundary. Consult [Fiber object ownership](https://r3f.docs.pmnd.rs/api/objects) when handling primitives or manual disposal.

Use Fiber pointer events and existing Drei controls where appropriate. Check event propagation, occlusion and pointer capture; for manual raycasting use canvas-relative coordinates. Keep essential actions available through accessible DOM controls, with a usable fallback when WebGL fails.

For GLTF animation, follow the existing mixer/Drei abstraction, stop owned actions on cleanup and use elapsed/delta time consistently. Preserve pause, seek and replay-clock semantics when animation represents match time. A reduced-motion setting may simplify decorative motion without altering factual playback.

## Verify the changed scene

Inspect the real browser interaction: mount/unmount, resize, selection, camera movement and affected animations. For a performance change, compare frame timing or renderer resource counts on the same scene. Run relevant frontend tests and build through root scripts. Report browser or GPU limitations honestly.
