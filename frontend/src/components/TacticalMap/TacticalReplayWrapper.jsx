// frontend/src/components/Dashboard/TacticalReplayWrapper.jsx
import React from 'react';
import Replay2DViewer from '../Stats/Replay2DViewer';
import { X, PlayCircle, AlertTriangle } from 'lucide-react';
import '../../styles/pages/coachDashboard.css';

const TacticalReplayWrapper = ({ scenario, onClose, externalControl }) => {
  if (!scenario) return null;

  return (
    <div className="tactical-replay-wrapper">
      <div className="replay-header">
        <div className="replay-title">
          <PlayCircle size={20} className="text-yellow-400" />
          <div>
            <h3>Análisis Táctico: {scenario.title}</h3>
            <span className="replay-timestamp">Round {scenario.round} • {scenario.timestamp}</span>
          </div>
        </div>
        <div className="replay-controls">
          <button className="close-replay-btn" onClick={onClose}>
            <X size={18} />
            <span>Salir del Modo Revisión</span>
          </button>
        </div>
      </div>

      <div className="replay-container-framed">
        <Replay2DViewer 
          matchId={scenario.matchId}
          initialRound={scenario.round}
          externalControl={externalControl}
          scenarioContext={{
            title: scenario.title,
            description: scenario.description
          }}
        />
        
        {/* Overlay for AI Annotation */}
        <div className="ai-annotation-overlay">
          <div className="annotation-card">
            <div className="annotation-header">
              <AlertTriangle size={16} className="text-red-400" />
              <span>Detección de Error</span>
            </div>
            <p>{scenario.description}</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TacticalReplayWrapper;
