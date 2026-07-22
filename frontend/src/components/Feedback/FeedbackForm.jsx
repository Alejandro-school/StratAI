import React, { useState } from 'react';
import { Bug, Lightbulb, Palette, HelpCircle, Send, CheckCircle, AlertTriangle } from 'lucide-react';
import { API_URL } from '../../utils/api';

const CATEGORIES = [
  { id: 'bug', label: 'Error', icon: Bug },
  { id: 'sugerencia', label: 'Sugerencia', icon: Lightbulb },
  { id: 'ux', label: 'UX / Diseño', icon: Palette },
  { id: 'otro', label: 'Otro', icon: HelpCircle },
];

const FeedbackForm = ({ onSubmitted }) => {
  const [category, setCategory] = useState('');
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);
  const [toast, setToast] = useState(null); // { type: 'success'|'error', text }

  const canSubmit = category && message.length >= 10 && message.length <= 2000 && !sending;

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!canSubmit) return;

    setSending(true);
    setToast(null);

    try {
      const res = await fetch(`${API_URL}/feedback`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ category, message }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.detail || `Error ${res.status}`);
      }

      setToast({ type: 'success', text: '¡Comentarios enviados! Gracias por ayudarnos a mejorar.' });
      setCategory('');
      setMessage('');
      onSubmitted?.();
    } catch (err) {
      setToast({ type: 'error', text: err.message || 'Error al enviar los comentarios.' });
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="fb-card">
      <h3 className="fb-card-title">
        <Send size={16} style={{ color: 'var(--p-accent)' }} />
        Enviar comentarios
      </h3>

      {toast && (
        <div className={`fb-toast ${toast.type}`}>
          {toast.type === 'success' ? <CheckCircle size={16} /> : <AlertTriangle size={16} />}
          {toast.text}
        </div>
      )}

      <form onSubmit={handleSubmit}>
        <div className="fb-categories">
          {CATEGORIES.map((cat) => {
            const Icon = cat.icon;
            return (
              <button
                key={cat.id}
                type="button"
                className={`fb-cat-pill cat-${cat.id} ${category === cat.id ? 'active' : ''}`}
                onClick={() => setCategory(cat.id)}
              >
                <Icon size={14} />
                {cat.label}
              </button>
            );
          })}
        </div>

        <div className="fb-textarea-wrap">
          <textarea
            className="fb-textarea"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder="Describe tus comentarios con el mayor detalle posible..."
            maxLength={2000}
          />
          <span className={`fb-char-count ${message.length > 2000 ? 'over' : ''}`}>
            {message.length} / 2000
          </span>
        </div>

        <button type="submit" className="fb-submit" disabled={!canSubmit}>
          {sending ? (
            <>
              <span className="fb-spinner" />
              Enviando...
            </>
          ) : (
            <>
              <Send size={14} />
              Enviar comentarios
            </>
          )}
        </button>
      </form>
    </div>
  );
};

export default FeedbackForm;
