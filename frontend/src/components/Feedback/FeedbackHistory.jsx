import React, { useEffect, useState } from 'react';
import { Clock, Inbox } from 'lucide-react';
import { API_URL } from '../../utils/api';

const STATUS_LABELS = {
  pending: 'Pendiente',
  reviewed: 'Revisado',
  resolved: 'Resuelto',
};

const CATEGORY_LABELS = {
  bug: 'Error',
  sugerencia: 'Sugerencia',
  ux: 'UX / Diseño',
  otro: 'Otro',
};

const formatDate = (iso) => {
  const d = new Date(iso);
  return d.toLocaleDateString('es-ES', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};

const FeedbackHistory = ({ refreshKey }) => {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const fetchHistory = async () => {
      setLoading(true);
      try {
        const res = await fetch(`${API_URL}/feedback`, {
          credentials: 'include',
        });
        if (!res.ok) throw new Error();
        const data = await res.json();
        if (!cancelled) setItems(data.feedback || []);
      } catch {
        if (!cancelled) setItems([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    fetchHistory();
    return () => { cancelled = true; };
  }, [refreshKey]);

  if (loading) {
    return (
      <div className="fb-card">
        <h3 className="fb-card-title">
          <Clock size={16} style={{ color: 'var(--p-accent)' }} />
          Tu historial
        </h3>
        <div className="fb-loading">
          <span className="fb-spinner" />
          Cargando historial...
        </div>
      </div>
    );
  }

  return (
    <div className="fb-card">
      <h3 className="fb-card-title">
        <Clock size={16} style={{ color: 'var(--p-accent)' }} />
        Tu historial
      </h3>

      {items.length === 0 ? (
        <div className="fb-empty">
          <Inbox size={40} className="fb-empty-icon" />
          <p>No has enviado comentarios todavía.</p>
        </div>
      ) : (
        <div className="fb-history-list">
          {items.map((item) => (
            <div key={item.id} className="fb-history-item">
              <span className={`fb-badge cat-${item.category}`}>
                {CATEGORY_LABELS[item.category] || item.category}
              </span>
              <div className="fb-history-body">
                <p className="fb-history-message">{item.message}</p>
                <div className="fb-history-date">{formatDate(item.created_at)}</div>
              </div>
              <span className={`fb-status ${item.status}`}>
                {STATUS_LABELS[item.status] || item.status}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default FeedbackHistory;
