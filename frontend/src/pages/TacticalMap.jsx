import React from 'react';
import { TacticalMapProvider } from '../context/TacticalMapContext';
import TacticalMapExperience from '../features/tactical-map/components/TacticalMapExperience';

const TacticalMap = () => (
  <TacticalMapProvider>
    <TacticalMapExperience />
  </TacticalMapProvider>
);

export default TacticalMap;
