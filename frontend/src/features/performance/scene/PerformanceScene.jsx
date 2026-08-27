import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import * as THREE from 'three';
import GeneralOverviewNode from './GeneralOverviewNode';
import MapNode from './MapNode';

const GENERAL_VIEW = {
  camera: new THREE.Vector3(0, 4.1, 17.2),
  target: new THREE.Vector3(0, 2, -3.4),
};

const createGalleryLayouts = (contexts, selectedId) => {
  const maps = contexts.slice(1);
  const selectedIndex = maps.findIndex(({ id }) => id === selectedId);
  const focusIndex = selectedIndex >= 0 ? selectedIndex : Math.floor((maps.length - 1) / 2);

  return Object.fromEntries(maps.map((context, index) => {
    const offset = index - focusIndex;
    const distance = Math.abs(offset);
    return [context.id, {
      index,
      position: [offset * 3.28, 2.18 - distance * 0.035, -4.25 - distance * 0.24],
      rotationY: -offset * 0.032,
      visible: distance <= 3,
    }];
  }));
};

const getView = (selectedId, layouts) => {
  const layout = layouts[selectedId];
  if (!layout) return GENERAL_VIEW;

  return {
    camera: new THREE.Vector3(
      layout.position[0],
      layout.position[1] + 1.2,
      layout.position[2] + 13.6,
    ),
    target: new THREE.Vector3(
      layout.position[0],
      layout.position[1] - 0.05,
      layout.position[2],
    ),
  };
};

const easeInOutCubic = (progress) => (
  progress < 0.5
    ? 4 * progress ** 3
    : 1 - ((-2 * progress + 2) ** 3) / 2
);

const CameraRig = ({ selectedId, layouts }) => {
  const { camera, pointer } = useThree();
  const transition = useRef(null);
  const settledPosition = useMemo(() => new THREE.Vector3(), []);
  const targetCamera = useMemo(() => new THREE.PerspectiveCamera(), []);

  useEffect(() => {
    const view = getView(selectedId, layouts);
    const midpoint = camera.position.clone().lerp(view.camera, 0.5);
    midpoint.y += 0.85;
    midpoint.z += 1.65;

    targetCamera.position.copy(view.camera);
    targetCamera.lookAt(view.target);

    transition.current = {
      elapsed: 0,
      duration: selectedId === 'general' ? 1.15 : 1.35,
      curve: new THREE.QuadraticBezierCurve3(
        camera.position.clone(),
        midpoint,
        view.camera.clone(),
      ),
      startQuaternion: camera.quaternion.clone(),
      endQuaternion: targetCamera.quaternion.clone(),
      view,
    };
  }, [camera, layouts, selectedId, targetCamera]);

  useFrame((_, delta) => {
    const current = transition.current;
    if (!current) return;

    if (current.elapsed < current.duration) {
      current.elapsed = Math.min(current.elapsed + delta, current.duration);
      const progress = easeInOutCubic(current.elapsed / current.duration);
      current.curve.getPoint(progress, camera.position);
      camera.quaternion.slerpQuaternions(
        current.startQuaternion,
        current.endQuaternion,
        progress,
      );
      return;
    }

    settledPosition.copy(current.view.camera);
    settledPosition.x += pointer.x * 0.08;
    settledPosition.y += pointer.y * 0.05;
    camera.position.lerp(settledPosition, 1 - Math.exp(-delta * 2.4));
  });

  return null;
};

const MapGallery = ({
  contexts,
  layouts,
  selectedId,
  comparisonPlayer,
  compareMode,
  onSelect,
}) => {
  const [hoveredId, setHoveredId] = useState(null);
  const handleHover = useCallback((contextId) => setHoveredId(contextId), []);
  const hoveredIndex = hoveredId ? layouts[hoveredId]?.index : null;

  return (
    <>
      {contexts.slice(1).map((context) => {
        const layout = layouts[context.id];
        if (!layout.visible) return null;

        const directionFromHover = hoveredIndex === null
          ? 0
          : Math.sign(layout.index - hoveredIndex);
        const position = [
          layout.position[0] + directionFromHover * 0.42,
          layout.position[1],
          layout.position[2],
        ];

        return (
          <MapNode
            key={context.id}
            context={context}
            position={position}
            rotationY={layout.rotationY}
            selected={selectedId === context.id}
            dimmed={selectedId !== 'general' && selectedId !== context.id}
            comparisonContext={comparisonPlayer?.contexts?.[context.id]}
            comparisonName={comparisonPlayer?.name}
            compareMode={compareMode}
            onHover={handleHover}
            onSelect={onSelect}
          />
        );
      })}
    </>
  );
};

const PerformanceScene = ({
  contexts,
  selectedId,
  comparisonPlayer,
  compareMode,
  onSelect,
}) => {
  const layouts = useMemo(
    () => createGalleryLayouts(contexts, selectedId),
    [contexts, selectedId],
  );

  return (
    <Canvas
      dpr={[1, 1.65]}
      camera={{ position: [0, 4.1, 17.2], fov: 38, near: 0.1, far: 80 }}
      gl={{ antialias: true, alpha: true, powerPreference: 'high-performance' }}
      onCreated={({ gl }) => gl.setClearColor(0x000000, 0)}
    >
      <hemisphereLight args={['#d5f5ff', '#17313d', 1.7]} />
      <ambientLight intensity={1.25} color="#cfedf5" />
      <directionalLight position={[4, 9, 6]} intensity={2.8} color="#effcff" />
      <pointLight position={[-8, 4, 2]} intensity={3.6} color="#3ba7df" distance={18} />
      <pointLight position={[8, 4, 2]} intensity={3.25} color="#5cf2cd" distance={18} />

      <MapGallery
        contexts={contexts}
        layouts={layouts}
        selectedId={selectedId}
        comparisonPlayer={comparisonPlayer}
        compareMode={compareMode}
        onSelect={onSelect}
      />
      {selectedId === 'general' && (
        <GeneralOverviewNode
          context={contexts[0]}
          comparison={comparisonPlayer?.contexts?.general}
          comparisonName={comparisonPlayer?.name}
          compareMode={compareMode}
        />
      )}
      <CameraRig selectedId={selectedId} layouts={layouts} />
    </Canvas>
  );
};

export default PerformanceScene;
