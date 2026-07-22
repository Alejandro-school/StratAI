import React from 'react';
import { AlertTriangle, Brain, Check, ClipboardPlus, PlayCircle, Send, Sparkles } from 'lucide-react';
import { CardRenderer } from '../CoachCards';
import EvidenceTray from './EvidenceTray';

const getVisibleText = (message) => {
  if (!message.isTyping || message.visibleChars === undefined) return message.text;
  return message.text.substring(0, message.visibleChars);
};

const CoachConversationPanel = ({
  messages,
  chatInput,
  setChatInput,
  isAiTyping,
  quickActions,
  chatEndRef,
  isPlaying,
  activeClip,
  evidenceList,
  selectedEvidence,
  analysisStatus,
  onSelectEvidence,
  onPlayEvidence,
  onTogglePlan,
  isSavedToPlan,
  onSubmitMessage,
  onPlayInteraction
}) => {
  const handleSubmit = (event) => {
    event.preventDefault();
    onSubmitMessage(chatInput);
  };

  return (
    <aside className="coach-conversation-panel">
      <div className="coach-conversation-header">
        <div className="coach-avatar">
          <Brain size={19} aria-hidden="true" />
        </div>
        <div className="coach-identity">
          <span className="analysis-kicker">Copiloto IA</span>
          <strong>StratAI Coach</strong>
          <span className="coach-model-status">Modelo en entrenamiento</span>
        </div>
        <Sparkles size={16} className="coach-header-spark" aria-hidden="true" />
      </div>

      <EvidenceTray
        evidenceList={evidenceList}
        selectedEvidence={selectedEvidence}
        status={analysisStatus}
        onSelectEvidence={onSelectEvidence}
        onPlayEvidence={onPlayEvidence}
      />

      {selectedEvidence && (
        <section className="coach-active-brief" aria-label="Hallazgo activo">
          <div className="coach-active-icon"><AlertTriangle size={16} aria-hidden="true" /></div>
          <div>
            <span>Foco actual · Ronda {selectedEvidence.round}</span>
            <strong>{selectedEvidence.title}</strong>
            <p>{selectedEvidence.recommendation}</p>
          </div>
          <div className="coach-active-actions">
            <button type="button" onClick={() => onPlayEvidence(selectedEvidence)}>
              <PlayCircle size={14} aria-hidden="true" /> Ver momento
            </button>
            <button type="button" className={isSavedToPlan ? 'saved' : ''} onClick={() => onTogglePlan(selectedEvidence.id)}>
              {isSavedToPlan ? <Check size={14} aria-hidden="true" /> : <ClipboardPlus size={14} aria-hidden="true" />}
              {isSavedToPlan ? 'Guardado' : 'Plan'}
            </button>
          </div>
        </section>
      )}

      <div className="coach-message-list" aria-label="Conversación con el coach">
        {messages.map((message) => (
          <div key={message.id} className={`coach-message ${message.sender}`}>
            {message.context && <span className="coach-message-context">{message.context}</span>}
            <p>{getVisibleText(message)}</p>

            {message.cards?.length > 0 && (
              <div className="coach-card-stack">
                {message.cards.map((card) => (
                  <CardRenderer key={card.id || card.title} card={card} />
                ))}
              </div>
            )}

            {message.interaction && !message.isTyping && (
              <button
                type="button"
                className={`coach-message-action ${isPlaying && activeClip?.startTick === message.interaction.startTick ? 'playing' : ''}`}
                onClick={() => onPlayInteraction(message.interaction)}
              >
                <PlayCircle size={14} aria-hidden="true" />
                Ver fragmento en 2D
              </button>
            )}
          </div>
        ))}

        {isAiTyping && (
          <div className="coach-typing-indicator">
            <span />
            <span />
            <span />
          </div>
        )}

        <div ref={chatEndRef} />
      </div>

      <div className="coach-quick-actions">
        {quickActions.map((action) => (
          <button key={action} type="button" onClick={() => onSubmitMessage(action)}>
            {action}
          </button>
        ))}
      </div>

      <span className="sr-only" aria-live="polite">
        {isAiTyping ? 'El coach está preparando una respuesta.' : ''}
      </span>

      <form className="coach-message-form" onSubmit={handleSubmit}>
        <label className="sr-only" htmlFor="coach-question">Pregunta sobre esta partida</label>
        <input
          id="coach-question"
          name="coach-question"
          type="text"
          value={chatInput}
          onChange={(event) => setChatInput(event.target.value)}
          placeholder="Pregunta sobre esta partida…"
          autoComplete="off"
        />
        <button type="submit" aria-label="Enviar mensaje">
          <Send size={18} aria-hidden="true" />
        </button>
      </form>
    </aside>
  );
};

export default CoachConversationPanel;
