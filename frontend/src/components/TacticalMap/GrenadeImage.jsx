import React from 'react';

const GRENADE_IMAGE_BY_TYPE = {
  smoke: '/images/weapons/weapon_smokegrenade.png',
  flash: '/images/weapons/weapon_flashbang.png',
  he: '/images/weapons/weapon_hegrenade.png',
  molotov: '/images/weapons/weapon_molotov.png',
  incendiary: '/images/weapons/weapon_incgrenade.png',
};

const GrenadeImage = ({ type, size = 24, className = '' }) => (
  <img
    src={GRENADE_IMAGE_BY_TYPE[type] ?? GRENADE_IMAGE_BY_TYPE.he}
    alt=""
    aria-hidden="true"
    width={size}
    height={size}
    className={`grenade-icon-img ${className}`}
    draggable="false"
  />
);

export default React.memo(GrenadeImage);
