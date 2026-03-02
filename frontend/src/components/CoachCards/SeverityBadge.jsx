import React from 'react';

const SeverityBadge = ({ severity = 'MEDIO' }) => {
  const normalized = severity.toUpperCase();
  return (
    <span className={`severity-badge severity-${normalized.toLowerCase()}`}>
      {normalized}
    </span>
  );
};

export default SeverityBadge;
