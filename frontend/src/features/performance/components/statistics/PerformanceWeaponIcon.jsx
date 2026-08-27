import React from 'react';
import {
  equipmentIconPath,
  equipmentLabel,
} from '../../../replay2d/domain/weaponPresentation';

const PerformanceWeaponIcon = ({ weapon, featured = false }) => {
  const source = equipmentIconPath(weapon);
  if (!source) return null;

  return (
    <img
      src={source}
      alt=""
      aria-hidden="true"
      title={equipmentLabel(weapon)}
      className={`pf3-weapon-icon ${featured ? 'is-featured' : ''}`}
    />
  );
};

export default PerformanceWeaponIcon;
