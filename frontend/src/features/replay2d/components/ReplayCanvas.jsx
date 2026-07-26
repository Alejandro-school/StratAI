import { LocateFixed, Minus, Plus } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { getMapPngFallback, getMapSource, MAP_LEVELS } from "../domain/replayConfig";
import { closestFrameIndex } from "../domain/replayModel";
import {
  playerIdentity,
  PROJECTILE_ICON_IDS,
  projectileIconPath,
} from "../domain/weaponPresentation";
import { renderReplayScene } from "../renderer/renderReplayScene";
import { createViewport, gameToMap, screenToWorld, zoomAtPoint } from "../renderer/replayViewport";

function useReducedMotion() {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    const media = window.matchMedia("(prefers-reduced-motion: reduce)");
    const update = () => setReduced(media.matches);
    update();
    media.addEventListener?.("change", update);
    return () => media.removeEventListener?.("change", update);
  }, []);
  return reduced;
}

function loadImage(source, onLoad) {
  const image = new Image();
  image.onload = () => onLoad(image);
  image.onerror = () => {
    if (image.src.endsWith(".webp")) image.src = getMapPngFallback(source);
  };
  image.src = source;
  return image;
}

let projectileIconCatalogPromise;

function preloadProjectileIconCatalog() {
  if (!projectileIconCatalogPromise) {
    projectileIconCatalogPromise = Promise.all(
      Object.keys(PROJECTILE_ICON_IDS).map((type) => new Promise((resolve) => {
        const image = new Image();
        image.onload = () => resolve([type, image]);
        image.onerror = () => resolve([type, null]);
        image.src = projectileIconPath(type);
      })),
    ).then((entries) => Object.fromEntries(entries.filter(([, image]) => image)));
  }
  return projectileIconCatalogPromise;
}

export function ReplayCanvas({
  mapName,
  config,
  frame,
  frames,
  events,
  tick,
  tickRate,
  layers,
  annotations,
  focusPlayerId,
  onFocusPlayer,
  levelMode,
  activeLevel,
  zThreshold,
  annotationTool,
  annotationColor,
  noteText,
  round,
  endTick,
  onCreateAnnotation,
  viewResetToken,
}) {
  const hostRef = useRef(null);
  const canvasRef = useRef(null);
  const hitTargets = useRef([]);
  const pointer = useRef(null);
  const [size, setSize] = useState({ width: 800, height: 600 });
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [follow, setFollow] = useState(false);
  const [draft, setDraft] = useState(null);
  const [images, setImages] = useState({});
  const [projectileIcons, setProjectileIcons] = useState({});
  const reducedMotion = useReducedMotion();
  const hasLevels = Boolean(MAP_LEVELS[mapName]);
  const viewport = useMemo(() => createViewport(size.width, size.height, zoom, pan), [pan, size, zoom]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return undefined;
    const observer = new ResizeObserver(([entry]) => {
      const { width, height } = entry.contentRect;
      if (width > 0 && height > 0) setSize({ width, height });
    });
    observer.observe(host);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    setImages({});
  }, [mapName]);

  useEffect(() => {
    const level = hasLevels ? activeLevel : "upper";
    const image = loadImage(getMapSource(mapName, level), (loaded) => {
      setImages((current) => ({ ...current, [level]: loaded }));
    });
    return () => {
      image.onload = null;
      image.onerror = null;
    };
  }, [activeLevel, hasLevels, mapName]);

  useEffect(() => {
    let disposed = false;
    preloadProjectileIconCatalog().then((catalog) => {
      if (!disposed) setProjectileIcons(catalog);
    });
    return () => {
      disposed = true;
    };
  }, []);

  useEffect(() => {
    setZoom(1);
    setPan({ x: 0, y: 0 });
    setFollow(false);
  }, [mapName, round, viewResetToken]);

  useEffect(() => {
    if (!follow || !focusPlayerId || !frame) return;
    const player = frame.players?.find((item) => playerIdentity(item) === String(focusPlayerId));
    if (!player) return;
    const map = gameToMap(player, config);
    const mapScale = viewport.baseScale * zoom;
    setPan({
      x: (512 - map.x) * mapScale,
      y: (512 - map.y) * mapScale,
    });
  }, [config, focusPlayerId, follow, frame, viewport.baseScale, zoom]);

  const trail = useMemo(() => {
    if (!focusPlayerId || !frames?.length) return [];
    const currentIndex = closestFrameIndex(frames, tick);
    const samples = Math.max(1, Math.round((tickRate * 5) / Math.max(1, frames[1]?.tick - frames[0]?.tick || 1)));
    return frames.slice(Math.max(0, currentIndex - samples), currentIndex + 1)
      .map((sample) => sample.players?.find((player) => playerIdentity(player) === String(focusPlayerId)))
      .filter(Boolean)
      .map((player) => ({ x: player.x, y: player.y }));
  }, [focusPlayerId, frames, tick, tickRate]);

  useEffect(() => {
    if (!canvasRef.current || size.width < 1 || size.height < 1) return;
    hitTargets.current = renderReplayScene({
      canvas: canvasRef.current,
      size,
      viewport,
      images,
      projectileIcons,
      frame,
      events,
      annotations,
      draft,
      trail,
      config,
      tick,
      tickRate,
      layers,
      activeLevel,
      hasLevels,
      zThreshold,
      focusPlayerId,
      reducedMotion,
    });
  }, [
    activeLevel, annotations, config, draft, events, focusPlayerId, frame, hasLevels,
    images, layers, projectileIcons, reducedMotion, size, tick, tickRate, trail, viewport, zThreshold,
  ]);

  const localPoint = (event) => {
    const rect = canvasRef.current.getBoundingClientRect();
    return { x: event.clientX - rect.left, y: event.clientY - rect.top };
  };

  const commitAnnotation = async (annotation) => {
    setDraft(null);
    await onCreateAnnotation({
      round,
      start_tick: Math.round(tick),
      end_tick: Math.round(endTick),
      type: annotation.type,
      points: annotation.points,
      text: annotation.text || "",
      color: annotation.color,
    });
  };

  const onPointerDown = (event) => {
    const screen = localPoint(event);
    const world = screenToWorld(screen, config, viewport);
    event.currentTarget.setPointerCapture(event.pointerId);
    pointer.current = { screen, world, pan, moved: false };
    if (annotationTool === "note") {
      commitAnnotation({ type: "note", points: [world], text: noteText || "Nota táctica", color: annotationColor });
      pointer.current = null;
    } else if (annotationTool) {
      setDraft({ type: annotationTool, points: [world], color: annotationColor });
    }
  };

  const onPointerMove = (event) => {
    if (!pointer.current) return;
    const screen = localPoint(event);
    const distance = Math.hypot(screen.x - pointer.current.screen.x, screen.y - pointer.current.screen.y);
    pointer.current.moved ||= distance > 3;
    if (!annotationTool) {
      setPan({
        x: pointer.current.pan.x + screen.x - pointer.current.screen.x,
        y: pointer.current.pan.y + screen.y - pointer.current.screen.y,
      });
      setFollow(false);
      return;
    }
    if (annotationTool === "note") return;
    const world = screenToWorld(screen, config, viewport);
    setDraft((current) => {
      if (!current) return current;
      if (annotationTool === "freehand") return { ...current, points: [...current.points, world] };
      return { ...current, points: [pointer.current.world, world] };
    });
  };

  const onPointerUp = (event) => {
    const state = pointer.current;
    pointer.current = null;
    event.currentTarget.releasePointerCapture(event.pointerId);
    if (annotationTool && annotationTool !== "note" && draft?.points?.length > 1) {
      commitAnnotation(draft);
      return;
    }
    setDraft(null);
    if (!state?.moved && !annotationTool) {
      const screen = localPoint(event);
      const hit = hitTargets.current.find((target) => Math.hypot(target.x - screen.x, target.y - screen.y) <= target.radius);
      if (hit) onFocusPlayer(String(hit.id));
    }
  };

  const changeZoom = (nextZoom, point = { x: size.width / 2, y: size.height / 2 }) => {
    const next = zoomAtPoint(viewport, point, nextZoom);
    setZoom(next.zoom);
    setPan(next.pan);
    setFollow(false);
  };

  return (
    <div className={`r2-canvas-host ${annotationTool ? "drawing" : ""}`} ref={hostRef}>
      <canvas
        ref={canvasRef}
        className="r2-canvas"
        style={{ width: size.width, height: size.height }}
        role="img"
        aria-label={`Mapa táctico de la ronda ${round}`}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={() => {
          pointer.current = null;
          setDraft(null);
        }}
        onWheel={(event) => {
          event.preventDefault();
          changeZoom(zoom * (event.deltaY > 0 ? 0.88 : 1.14), localPoint(event));
        }}
      />
      <div className="r2-viewport-controls">
        <button type="button" onClick={() => changeZoom(zoom * 1.2)} aria-label="Acercar"><Plus size={15} /></button>
        <span>{Math.round(zoom * 100)}%</span>
        <button type="button" onClick={() => changeZoom(zoom / 1.2)} aria-label="Alejar"><Minus size={15} /></button>
        <button type="button" className={follow ? "active" : ""} onClick={() => setFollow((value) => !value)} disabled={!focusPlayerId} aria-label="Seguir jugador seleccionado" aria-pressed={follow}>
          <LocateFixed size={15} />
        </button>
      </div>
      {hasLevels && <span className="r2-level-badge">{levelMode === "auto" ? "AUTO · " : ""}{activeLevel === "upper" ? "SUPERIOR" : "INFERIOR"}</span>}
    </div>
  );
}
