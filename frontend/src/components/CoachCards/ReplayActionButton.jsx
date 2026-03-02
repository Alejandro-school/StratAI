import React from 'react';
import { PlayCircle } from 'lucide-react';

const ReplayActionButton = ({ label = 'Ver en Replay', onClick }) => {
  return (
    <button type="button" className="coach-card-action" onClick={onClick}>
      <PlayCircle size={15} />
      {label}
    </button>
  );
};

export default ReplayActionButton;
