// frontend/src/components/Stats/common/EmptyState.jsx
import React from "react";
import { Inbox } from "lucide-react";
import "../../../styles/Stats/common.css";

/**
 * EmptyState - Componente para mostrar cuando no hay datos disponibles
 * 
 * @param {React.ReactNode} icon - Icono opcional (por defecto Inbox)
 * @param {string} title - Título del estado vacío
 * @param {string} description - Descripción o mensaje explicativo
 * @param {React.ReactNode} action - Botón u acción opcional
 */
export default function EmptyState({ 
  icon, 
  title = "No hay datos disponibles", 
  description = "No hay información para mostrar en este momento.",
  action 
}) {
  const Icon = icon || Inbox;
  
  return (
    <div className="cm-empty-state">
      <div className="cm-empty-icon">
        <Icon size={48} />
      </div>
      <h3 className="cm-empty-title">{title}</h3>
      <p className="cm-empty-description">{description}</p>
      {action && <div className="cm-empty-action">{action}</div>}
    </div>
  );
}

/**
 * EJEMPLO DE USO:
 * 
 * import EmptyState from './common/EmptyState';
 * import { FileQuestion } from 'lucide-react';
 * 
 * <EmptyState 
 *   icon={FileQuestion}
 *   title="No hay partidas"
 *   description="Sube demos para ver tus estadísticas"
 *   action={
 *     <button className="action-btn action-btn-primary">
 *       Subir Demo
 *     </button>
 *   }
 * />
 */


