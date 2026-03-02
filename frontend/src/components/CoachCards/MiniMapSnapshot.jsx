import React from 'react';

const renderMarker = (marker, index) => (
  <div
    key={`${marker.label}-${index}`}
    className={`mini-map-marker marker-${marker.variant || 'neutral'}`}
    style={{ left: `${marker.x}%`, top: `${marker.y}%` }}
    title={marker.label}
  >
    <span>{marker.label}</span>
  </div>
);

const MiniMapSnapshot = ({ mapName = 'de_mirage', markers = [] }) => {
  return (
    <div
      className="mini-map-snapshot"
      style={{ backgroundImage: `url('/maps/${mapName}.jpg')` }}
    >
      <div className="mini-map-overlay" />
      {markers.map(renderMarker)}
    </div>
  );
};

export default MiniMapSnapshot;
