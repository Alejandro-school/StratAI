import React from 'react';

const ContextRail = ({ contexts, selectedId, onSelect }) => (
  <div className="pf3-context-rail" role="tablist" aria-label="Contexto de mapa">
    {contexts.map((context, index) => {
      const selected = context.id === selectedId;
      return (
        <button
          key={context.id}
          type="button"
          role="tab"
          aria-selected={selected}
          className={`pf3-context-chip ${selected ? 'is-selected' : ''}`}
          onClick={() => onSelect(context.id)}
        >
          <span className="pf3-context-index" aria-hidden="true">
            {String(index).padStart(2, '0')}
          </span>
          <span className="pf3-context-copy">
            <strong>{context.id === 'general' ? 'Todos los mapas' : context.name}</strong>
            <small>{selected ? 'Activo' : 'Abrir análisis'}</small>
          </span>
        </button>
      );
    })}
  </div>
);

export default ContextRail;
