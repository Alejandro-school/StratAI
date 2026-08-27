import React, { useMemo } from 'react';
import { useGLTF } from '@react-three/drei';
import * as THREE from 'three';
import { clone as cloneSkeleton } from 'three/examples/jsm/utils/SkeletonUtils.js';

const MODEL_CONFIG = {
  ct: {
    path: '/images/Landing/CT_model.glb',
    animationNames: ['ct_loadout_pistol01_walkup'],
  },
  t: {
    path: '/images/Landing/Tmodel-v1.glb',
    animationNames: [
      'tm_professional_varf_balkanidle_balkan_idle',
      'ct_loadout_pistol01_walkup',
    ],
  },
};

const DRACO_DECODER_PATH = '/draco/';

const applyStaticPose = (model, animations, animationNames) => {
  const clip = animationNames
    .map((name) => THREE.AnimationClip.findByName(animations, name))
    .find(Boolean) ?? animations[0];

  if (!clip) return;

  const mixer = new THREE.AnimationMixer(model);
  mixer.clipAction(clip).play();
  mixer.setTime(clip.duration * 0.18);
};

const normalizeModel = (model, targetHeight) => {
  const bounds = new THREE.Box3().setFromObject(model);
  const size = bounds.getSize(new THREE.Vector3());
  const scale = targetHeight / size.y;
  model.scale.setScalar(scale);

  const scaledBounds = new THREE.Box3().setFromObject(model);
  const center = scaledBounds.getCenter(new THREE.Vector3());
  model.position.set(-center.x, -scaledBounds.min.y, -center.z);
};

const StaticFactionModel = ({
  side,
  position,
  rotation,
  height = 4.8,
  active = true,
  onSelect,
}) => {
  const config = MODEL_CONFIG[side];
  const { scene, animations } = useGLTF(config.path, DRACO_DECODER_PATH);

  const model = useMemo(() => {
    const clone = cloneSkeleton(scene);
    applyStaticPose(clone, animations, config.animationNames);
    normalizeModel(clone, height);

    clone.traverse((child) => {
      if (!child.isMesh) return;
      child.castShadow = true;
      child.receiveShadow = true;
    });

    return clone;
  }, [animations, config.animationNames, height, scene]);

  return (
    <group
      position={position}
      rotation={rotation}
      scale={active ? 1 : 0.985}
      onClick={(event) => {
        event.stopPropagation();
        onSelect?.(side);
      }}
      onPointerOver={() => {
        document.body.style.cursor = 'pointer';
      }}
      onPointerOut={() => {
        document.body.style.cursor = 'default';
      }}
    >
      <primitive object={model} />
    </group>
  );
};

useGLTF.preload(MODEL_CONFIG.ct.path, DRACO_DECODER_PATH);
useGLTF.preload(MODEL_CONFIG.t.path, DRACO_DECODER_PATH);

export default StaticFactionModel;
