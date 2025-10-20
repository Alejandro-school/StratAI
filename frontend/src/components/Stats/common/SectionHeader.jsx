// frontend/src/components/Stats/common/SectionHeader.jsx
import React from "react";
import "../../../styles/Stats/common.css";

/**
 * SectionHeader - Componente reutilizable para encabezados de secciones
 * 
 * @param {string} title - Título de la sección
 * @param {string} description - Descripción breve de la sección
 * @param {React.ReactNode} actions - Botones u otros elementos opcionales de acción
 */
export default function SectionHeader({ title, description, actions }) {
  return (
    <div className="cm-section-header">
      <div className="cm-header-left">
        <h2 className="cm-section-title">{title}</h2>
        {description && <p className="cm-section-description">{description}</p>}
      </div>
      {actions && <div className="cm-header-actions">{actions}</div>}
    </div>
  );
}

/**
 * EJEMPLO DE USO:
 * 
 * import SectionHeader from './common/SectionHeader';
 * 
 * <SectionHeader 
 *   title="Desempeño Personal" 
 *   description="Resumen de tu rendimiento global"
 *   actions={
 *     <button className="action-btn action-btn-primary">
 *       Filtrar
 *     </button>
 *   }
 * />
 */


