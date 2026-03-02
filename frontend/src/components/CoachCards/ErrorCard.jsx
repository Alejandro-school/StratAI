import React from 'react';
import { AlertTriangle } from 'lucide-react';
import SeverityBadge from './SeverityBadge';
import ReplayActionButton from './ReplayActionButton';
import MiniMapSnapshot from './MiniMapSnapshot';

const ErrorCard = ({ card, onOpenReview }) => {
  const { title, detail, impactTag, severity, mapName, markers = [], replayAction } = card;

  return (
    <article className="coach-card coach-card-error">
      <div className="coach-card-header">
        <div className="coach-card-title-wrap">
          <AlertTriangle size={16} />
          <h4>{title}</h4>
        </div>
        <SeverityBadge severity={severity} />
      </div>

      <p className="coach-card-detail">{detail}</p>
      {impactTag && <span className="coach-card-tag">{impactTag}</span>}

      <MiniMapSnapshot mapName={mapName} markers={markers} />

      <ReplayActionButton onClick={() => onOpenReview?.(replayAction)} />
    </article>
  );
};

export default ErrorCard;
