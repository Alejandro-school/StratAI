import React from 'react';
import { FlaskConical } from 'lucide-react';
import ReplayActionButton from './ReplayActionButton';
import MiniMapSnapshot from './MiniMapSnapshot';

const UtilityCard = ({ card, onOpenReview }) => {
  const {
    title,
    detail,
    enemiesBlinded = 0,
    alliesBlinded = 0,
    effectiveness = 0,
    mapName,
    markers = [],
    replayAction
  } = card;

  return (
    <article className="coach-card coach-card-utility">
      <div className="coach-card-header">
        <div className="coach-card-title-wrap">
          <FlaskConical size={16} />
          <h4>{title}</h4>
        </div>
        <span className="coach-card-tag">{effectiveness}% efectiva</span>
      </div>

      <p className="coach-card-detail">{detail}</p>

      <MiniMapSnapshot mapName={mapName} markers={markers} />

      <div className="utility-outcomes">
        <span>Enemigos cegados: <strong>{enemiesBlinded}</strong></span>
        <span>Aliados cegados: <strong>{alliesBlinded}</strong></span>
      </div>

      <ReplayActionButton onClick={() => onOpenReview?.(replayAction)} />
    </article>
  );
};

export default UtilityCard;
