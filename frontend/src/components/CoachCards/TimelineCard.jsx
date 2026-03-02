import React from 'react';
import { ListChecks } from 'lucide-react';
import ReplayActionButton from './ReplayActionButton';

const TimelineCard = ({ card, onOpenReview }) => {
  const { title, detail, events = [], replayAction } = card;

  return (
    <article className="coach-card coach-card-timeline">
      <div className="coach-card-header">
        <div className="coach-card-title-wrap">
          <ListChecks size={16} />
          <h4>{title}</h4>
        </div>
      </div>

      <p className="coach-card-detail">{detail}</p>

      <div className="timeline-strip">
        {events.map((event) => (
          <button
            key={`${event.timestamp}-${event.label}`}
            type="button"
            className={`timeline-event event-${event.type}`}
            title={`${event.timestamp} · ${event.label}`}
            onClick={() => onOpenReview?.({ ...replayAction, timestamp: event.timestamp })}
          >
            <span>{event.timestamp}</span>
          </button>
        ))}
      </div>

      <ReplayActionButton onClick={() => onOpenReview?.(replayAction)} />
    </article>
  );
};

export default TimelineCard;
