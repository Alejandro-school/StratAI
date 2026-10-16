import React, { Suspense, useEffect, useMemo, useRef } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { useGLTF } from '@react-three/drei';
import * as THREE from 'three';
import { clone as cloneSkeleton } from 'three/examples/jsm/utils/SkeletonUtils.js';
import { KTX2Loader } from 'three/examples/jsm/loaders/KTX2Loader.js';
import { FEATURES, getFeature, writeJourneyPose } from './journey';
import { AGENTS, UTILITY_ARC_POINTS, utilityPanelOcclusion } from './sceneLayout';
import { layoutWorldPins } from './pinsLayout';
import { BASELINE_SCENE, MEASURE_SCENE, ScenePerformance } from './ScenePerformance';
import { animationDelta, playbackDelta, SceneRendering } from './SceneRendering';
import { flightEase, writeExplorationPose } from './cameraNavigation';
import { transitionEase, TOOL_CAMERA_MS, writeEntryPose, writeToolPose } from './sceneTransitions';

function SceneAssets({ onReady, selected, reducedMotion, playback, activity, stage }) {
  const { gl } = useThree();
  const ktx = useMemo(() => new KTX2Loader().setTranscoderPath('/basis/').setWorkerLimit(2).detectSupport(gl), [gl]);
  useEffect(() => () => ktx.dispose(), [ktx]);
  // Own the worker pool above the boundary: initial suspension must not throw
  // away the loader that is currently decoding the cached asset.
  return <Suspense fallback={null}>
    <Mirage onReady={onReady} ktx={ktx} stage={stage} />
    <Agent configuration={AGENTS.ct} />
    <Agent configuration={AGENTS.t} />
    {FEATURES.map(feature => <Beacon key={feature.id} feature={feature} selected={selected} reducedMotion={reducedMotion} />)}
    <Demonstration selected={selected} playback={playback} activity={activity} />
  </Suspense>;
}

function Mirage({ onReady, ktx, stage }) {
  const { gl } = useThree();
  const asset = useGLTF(BASELINE_SCENE ? '/landing-world/mirage-textured.glb' : '/landing-world/mirage-exploration.glb', '/draco/', true, loader => {
    loader.setKTX2Loader(ktx);
    loader.dracoLoader?.setWorkerLimit(2);
  });
  const model = useMemo(() => {
    const scene = asset.scene.clone(true);
    scene.traverse(object => {
      if (!object.isMesh) return;
      object.castShadow = true;
      object.receiveShadow = true;
    });
    return scene;
  }, [asset.scene]);
  const levels = useMemo(() => ({ high: model.getObjectByName('site_high'), distant: model.getObjectByName('site_distant'), close: undefined, center: new THREE.Vector3(3, 0, 26) }), [model]);
  useFrame(({ camera, invalidate }) => {
    // Switch while A is still small on screen; never simplify it during close
    // exploration. Hysteresis avoids flicker when hovering at the threshold.
    const distance = camera.position.distanceTo(levels.center);
    const close = levels.close ? distance < 78 : distance < 68;
    if (close === levels.close) return;
    levels.close = close;
    if (levels.high) levels.high.visible = close;
    if (levels.distant) levels.distant.visible = !close;
    gl.shadowMap.needsUpdate = true;
    if (stage.current) stage.current.dataset.detail = close ? 'original-A' : 'distant';
    invalidate();
  }, -.5);
  useEffect(() => {
    gl.shadowMap.needsUpdate = true;
    onReady();
  }, [gl, onReady]);
  return <primitive object={model} dispose={null} />;
}

function Agent({ configuration }) {
  const asset = useGLTF(configuration.path, '/draco/');
  const model = useMemo(() => {
    const scene = cloneSkeleton(asset.scene);
    const mixer = new THREE.AnimationMixer(scene);
    const clip = asset.animations.find(clip => clip.name === configuration.clip);
    if (clip) {
      const action = mixer.clipAction(clip).play();
      mixer.setTime(clip.duration * configuration.pose);
      action.paused = true;
    }
    scene.updateMatrixWorld(true);
    scene.traverse(object => { if (object.isSkinnedMesh) object.computeBoundingBox(); });
    const box = new THREE.Box3().setFromObject(scene);
    const size = box.getSize(new THREE.Vector3());
    const center = box.getCenter(new THREE.Vector3());
    const scale = configuration.height / size.y;
    scene.position.sub(new THREE.Vector3(center.x, box.min.y, center.z));
    scene.traverse(object => { if (object.isMesh) object.castShadow = true; });
    return { scene, scale, mixer };
  }, [asset, configuration]);
  useEffect(() => () => { model.mixer.stopAllAction(); model.mixer.uncacheRoot(model.scene); }, [model]);
  return <group position={configuration.position} rotation={[0, configuration.rotation, 0]} scale={model.scale}>
    <primitive object={model.scene} dispose={null} />
  </group>;
}

function CameraRig({ journey, flight, navigation, interaction, onArrive, selected, pointer, reducedMotion, stage, activity, entry }) {
  const { camera, size, invalidate } = useThree();
  useEffect(() => {
    camera.fov = size.width < size.height ? 80 : 55;
    camera.updateProjectionMatrix();
  }, [camera, size.width, size.height]);
  const values = useMemo(() => {
    const initial = { position: new THREE.Vector3(), target: new THREE.Vector3(), right: new THREE.Vector3(), look: new THREE.Vector3(), flightPosition: new THREE.Vector3(), flightTarget: new THREE.Vector3(), fromPosition:new THREE.Vector3(), fromTarget:new THREE.Vector3(), entryPosition:new THREE.Vector3(), entryTarget:new THREE.Vector3(), offset:0, toolFresh:true, toolActive:false, wasEntry:false, frames: 0 };
    writeJourneyPose(0, initial.position, initial.look, false);
    return initial;
  }, []);
  useEffect(() => { values.toolFresh = true; invalidate(); }, [selected, size.width, size.height, invalidate, values]);
  const updateLens = () => {
    if (values.offset > .01) camera.setViewOffset(size.width, size.height, values.offset, 0, size.width, size.height);
    else camera.clearViewOffset();
    camera.updateProjectionMatrix();
  };
  useFrame((_, delta) => {
    const departure = entry?.current;
    if (departure?.active) {
      values.wasEntry = true;
      if (!departure.animate || reducedMotion) { activity.current.camera = false; return; }
      if (departure.fresh) {
        departure.fresh = false; values.entryPosition.copy(camera.position); values.entryTarget.copy(values.look);
        values.entryZoom = camera.zoom; values.entryOffset = values.offset;
      }
      const t = Math.min(1, (performance.now() - departure.startedAt) / departure.duration), ease = transitionEase(t);
      writeEntryPose(t, values.entryPosition, values.entryTarget, camera.position, values.look);
      camera.zoom = THREE.MathUtils.lerp(values.entryZoom, 1.8, ease);
      values.offset = THREE.MathUtils.lerp(values.entryOffset, 0, ease); updateLens(); camera.lookAt(values.look);
      activity.current.camera = t < 1;
      if (stage.current) { stage.current.dataset.sceneTransition = 'steam'; stage.current.dataset.transitionProgress = t.toFixed(3); stage.current.dataset.camera = camera.position.toArray().map(n => n.toFixed(2)).join(','); stage.current.dataset.zoom = camera.zoom.toFixed(3); }
      if (t < 1) invalidate();
      return;
    }
    if (values.wasEntry) { values.wasEntry = false; values.toolFresh = true; }
    const travel = flight.current;
    if (travel.running) {
      if (travel.fresh) { travel.fresh = false; travel.startZoom = camera.zoom; travel.startOffset = values.offset; values.flightPosition.copy(camera.position); values.flightTarget.copy(values.look); }
      else travel.elapsed += Math.min(delta, .1);
      const t = reducedMotion ? 1 : Math.min(1, travel.elapsed / 3.4);
      journey.current = THREE.MathUtils.lerp(travel.from, travel.to, flightEase(t));
      writeJourneyPose(journey.current * .82, values.position, values.target, false);
      const join = THREE.MathUtils.smoothstep(t, 0, .24);
      camera.zoom = THREE.MathUtils.lerp(travel.startZoom, 1, join); values.offset = THREE.MathUtils.lerp(travel.startOffset, 0, join); updateLens();
      camera.position.copy(values.flightPosition).lerp(values.position, join);
      values.look.copy(values.flightTarget).lerp(values.target, join); camera.lookAt(values.look);
      activity.current.camera = true;
      if (stage.current) { stage.current.dataset.travel = journey.current.toFixed(3); stage.current.dataset.camera = camera.position.toArray().map(n => n.toFixed(2)).join(','); }
      if (t === 1) { travel.running = false; onArrive(travel.to); }
      invalidate(); return;
    }
    const feature = getFeature(selected);
    if (feature) {
      values.position.fromArray(feature.camera);
      values.target.fromArray(feature.target);
      if (window.innerWidth > 760 && selected !== 'utility') {
        values.right.set(values.position.z - values.target.z, 0, values.target.x - values.position.x).normalize();
        values.target.addScaledVector(values.right, 3);
      }
      values.right.set(values.position.z - values.target.z, 0, values.target.x - values.position.x).normalize();
      if (values.toolFresh) {
        values.toolFresh = false; values.toolActive = true; values.toolStartedAt = performance.now();
        values.fromPosition.copy(camera.position); values.fromTarget.copy(values.look);
        values.toolZoom = camera.zoom; values.toolOffset = values.offset; values.toolDirection = Math.sign(values.target.x - values.look.x) || 1;
      }
    } else {
      values.toolActive = false;
      if (journey.current === 1) writeExplorationPose(navigation.current, values.position, values.target, reducedMotion || interaction.current ? undefined : pointer.current);
      else writeJourneyPose(0, values.position, values.target, reducedMotion);
      if (!reducedMotion && journey.current === 0) {
        const depth = 2 * (1 - journey.current) + .2;
        values.position.x += pointer.current.x * depth;
        values.position.y += pointer.current.y * depth * .4;
      }
    }
    const speed = reducedMotion ? 1 : 1 - Math.exp(-animationDelta(delta) * 3.5);
    const targetZoom = !feature && journey.current === 1 ? navigation.current.zoom : 1;
    const targetOffset = selected === 'utility' ? utilityPanelOcclusion(size.width) / 2 : 0;
    if (feature && values.toolActive) {
      const t = reducedMotion ? 1 : Math.min(1, (performance.now() - values.toolStartedAt) / TOOL_CAMERA_MS), ease = transitionEase(t);
      writeToolPose(t, values.fromPosition, values.fromTarget, values.position, values.target, camera.position, values.look, values.right, values.toolDirection);
      camera.zoom = THREE.MathUtils.lerp(values.toolZoom, targetZoom, ease); values.offset = THREE.MathUtils.lerp(values.toolOffset, targetOffset, ease);
      values.toolActive = t < 1;
    } else {
      camera.zoom = THREE.MathUtils.lerp(camera.zoom, targetZoom, speed); values.offset = THREE.MathUtils.lerp(values.offset, targetOffset, speed);
      camera.position.lerp(values.position, speed); values.look.lerp(values.target, speed);
    }
    updateLens();
    camera.lookAt(values.look);
    const moving = values.toolActive || camera.position.distanceToSquared(values.position) > .000001 || values.look.distanceToSquared(values.target) > .000001 || Math.abs(camera.zoom - targetZoom) > .00001 || Math.abs(values.offset - targetOffset) > .01;
    activity.current.camera = moving || interaction.current;
    if (moving) invalidate();
    if ((values.toolActive || ++values.frames % 25 === 0 || !moving) && stage.current) {
      stage.current.dataset.sceneTransition = values.toolActive ? 'tool' : 'idle';
      stage.current.dataset.camera = camera.position.toArray().map(value => value.toFixed(2)).join(',');
      stage.current.dataset.travel = journey.current.toFixed(3);
      stage.current.dataset.navigation = JSON.stringify(navigation.current);
    }
  }, -1);
  return null;
}

function Beacon({ feature, selected, reducedMotion }) {
  const beam = useRef();
  useFrame(({ clock }) => {
    if (beam.current && !reducedMotion) beam.current.scale.y = .94 + Math.sin(clock.elapsedTime * 1.7) * .06;
  });
  return <group position={feature.position} visible={!selected || selected === feature.id}>
    <group ref={beam}>
      <mesh position={[0, -1.3, 0]}><cylinderGeometry args={[.045, .045, 2.4, 8]} /><meshBasicMaterial color={feature.color} transparent opacity={.6} /></mesh>
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -2.48, 0]}><ringGeometry args={[.48, .53, 48]} /><meshBasicMaterial color={feature.color} side={THREE.DoubleSide} transparent opacity={.85} /></mesh>
      <mesh><sphereGeometry args={[.12, 16, 12]} /><meshBasicMaterial color={feature.color} /></mesh>
    </group>
  </group>;
}

function ProjectPins({ elements, selected, entered }) {
  const { camera, size } = useThree();
  const projection = useMemo(() => ({ point: new THREE.Vector3(), matrix: new THREE.Matrix4(), lens: new THREE.Matrix4(), width: 0, height: 0, selection: undefined, entered: undefined, points: FEATURES.map(feature => ({ id: feature.id, x: 0, y: 0, worldX: 0, worldY: 0 })) }), []);
  useFrame(() => {
    camera.updateMatrixWorld();
    if (projection.matrix.equals(camera.matrixWorld) && projection.lens.equals(camera.projectionMatrix) && projection.width === size.width && projection.height === size.height && projection.selection === selected && projection.entered === entered) return;
    projection.lens.copy(camera.projectionMatrix);
    projection.matrix.copy(camera.matrixWorld); projection.width = size.width; projection.height = size.height; projection.selection = selected; projection.entered = entered;
    for (let index = 0; index < FEATURES.length; index++) {
      projection.point.fromArray(FEATURES[index].position).project(camera);
      const point = projection.points[index];
      point.x = point.worldX = (projection.point.x * .5 + .5) * size.width;
      point.y = point.worldY = (-projection.point.y * .5 + .5) * size.height;
    }
    for (const point of layoutWorldPins(projection.points, size.width, size.height)) {
      const element = elements.current[point.id];
      if (!element?.button || !element.line) continue;
      element.button.style.left = `${point.x}px`; element.button.style.top = `${point.y}px`;
      element.line.setAttribute('x1', String(point.worldX)); element.line.setAttribute('y1', String(point.worldY));
      element.line.setAttribute('x2', String(point.x)); element.line.setAttribute('y2', String(point.y));
      element.line.style.opacity = entered && (!selected || selected === point.id) ? '.5' : '0';
    }
  });
  return null;
}

const arc = new THREE.CatmullRomCurve3(UTILITY_ARC_POINTS.map(point => new THREE.Vector3(...point)));

function Demonstration({ selected, playback, activity }) {
  const { invalidate } = useThree();
  const wasPlaying = useRef(false);
  const ball = useRef();
  const smoke = useRef();
  const route = useMemo(() => new THREE.TubeGeometry(arc, 60, .025, 6, false), []);
  const particles = useMemo(() => Array.from({ length: 15 }, (_, i) => ({ position: [8 + Math.sin(i * 2.4) * 1.6, .5 + (i % 4) * .5, 25 + Math.cos(i * 2.4) * 1.6], scale: .7 + (i % 3) * .18 })), []);
  useEffect(() => () => route.dispose(), [route]);
  useFrame((_, delta) => {
    const state = playback.current;
    if (state.playing) {
      state.progress = Math.min(1, state.progress + playbackDelta(delta, wasPlaying.current) / (selected === 'utility' ? 5 : 12));
      if (state.progress === 1) state.playing = false;
      if (state.playing) invalidate();
    }
    wasPlaying.current = state.playing;
    activity.current.playback = state.playing;
    if (ball.current) { arc.getPoint(Math.min(1, state.progress * 1.4), ball.current.position); ball.current.visible = selected === 'utility' && state.progress > 0 && state.progress < .72; }
    if (smoke.current) {
      const growth = THREE.MathUtils.smoothstep(state.progress, .62, 1);
      smoke.current.visible = selected === 'utility' && growth > 0;
      smoke.current.scale.setScalar(growth);
    }
  });
  return <group>
    <mesh geometry={route} visible={selected === 'utility'}><meshBasicMaterial color="#ffc48b" transparent opacity={.85} /></mesh>
    <mesh ref={ball} visible={false}><sphereGeometry args={[.13, 12, 10]} /><meshBasicMaterial color="#fff0c9" /></mesh>
    <group ref={smoke} visible={false} position={[8, 0, 25]}>{particles.map((particle, index) => <mesh key={index} position={[particle.position[0] - 8, particle.position[1], particle.position[2] - 25]} scale={particle.scale}>
      <sphereGeometry args={[1, 16, 12]} /><meshStandardMaterial color="#76898b" transparent opacity={.62} roughness={1} depthWrite={false} />
    </mesh>)}</group>
  </group>;
}

function Dust({ reducedMotion }) {
  const points = useRef();
  const positions = useMemo(() => {
    const array = new Float32Array(120 * 3);
    for (let i = 0; i < 120; i++) { array[i * 3] = Math.sin(i * 127.1) * 43; array[i * 3 + 1] = 2 + (i % 17); array[i * 3 + 2] = Math.cos(i * 71.7) * 40; }
    return array;
  }, []);
  useFrame((_, delta) => { if (points.current && !reducedMotion) points.current.rotation.y += Math.min(delta, .05) * .004; });
  return <points ref={points}>
    <bufferGeometry><bufferAttribute attach="attributes-position" array={positions} count={120} itemSize={3} /></bufferGeometry>
    <pointsMaterial color="#ffe2b3" size={.055} transparent opacity={.35} sizeAttenuation depthWrite={false} />
  </points>;
}

export default function WorldScene({ journey, flight, navigation, interaction, onArrive, selected, pointer, onReady, reducedMotion, playback, stage, entered, pins, renderRequest, entry }) {
  const activity = useRef({ camera: false, playback: false });
  return <Canvas shadows frameloop={BASELINE_SCENE ? 'always' : 'demand'} dpr={[1, 1.5]} camera={{ position: [75, 118, 90], fov: 55, near: .15, far: 600 }}
    gl={{ antialias: true, alpha: false, powerPreference: 'high-performance' }}
    onCreated={({ gl, camera }) => {
      gl.setClearColor('#0b191c'); gl.toneMappingExposure = .95;
      gl.shadowMap.type = THREE.PCFSoftShadowMap; gl.shadowMap.autoUpdate = false;
      const initialTarget = new THREE.Vector3();
      writeJourneyPose(journey.current, camera.position, initialTarget, reducedMotion);
      camera.lookAt(initialTarget);
    }}>
    <fog attach="fog" args={['#0b191c', 125, 300]} />
    <ambientLight intensity={.35} color="#e1ece6" />
    <hemisphereLight args={['#b6dcdf', '#66523b', .7]} />
    <directionalLight position={[-38, 80, 35]} color="#ffe0aa" intensity={2} castShadow shadow-mapSize={[2048, 2048]}
      shadow-camera-left={-72} shadow-camera-right={72} shadow-camera-top={72} shadow-camera-bottom={-72} shadow-camera-near={1} shadow-camera-far={220} shadow-normalBias={.04} shadow-bias={-.0001} />
    <directionalLight position={[35, 20, -35]} color="#a0d9ec" intensity={.5} />
    <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -4.4, 0]} receiveShadow><planeGeometry args={[700, 700]} /><meshStandardMaterial color="#102326" roughness={.94} /></mesh>
    <SceneAssets onReady={onReady} selected={entry?.current.active ? 'steam-entry' : selected} reducedMotion={reducedMotion} playback={playback} activity={activity} stage={stage} />
    <Dust reducedMotion={reducedMotion} />
    <CameraRig journey={journey} flight={flight} navigation={navigation} interaction={interaction} onArrive={onArrive} selected={selected} pointer={pointer} reducedMotion={reducedMotion} stage={stage} activity={activity} entry={entry} />
    <ProjectPins elements={pins} selected={selected} entered={entered} />
    <SceneRendering renderRequest={renderRequest} activity={activity} adaptive={!BASELINE_SCENE} />
    {MEASURE_SCENE && <ScenePerformance stage={stage} />}
  </Canvas>;
}
