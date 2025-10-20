// frontend/src/components/Stats/common/ErrorState.jsx
import React from "react";
import { AlertCircle } from "lucide-react";
import "../../../styles/Stats/common.css";

/**
 * ErrorState - Componente para mostrar errores al usuario
 * 
 * @param {string} message - Mensaje de error a mostrar
 * @param {Function} onRetry - Función callback para reintentar
 * @param {string} retryLabel - Texto del botón de reintentar
 */
export default function ErrorState({ 
  message = "Ha ocurrido un error al cargar los datos.", 
  onRetry,
  retryLabel = "Reintentar"
}) {
  return (
    <div className="cm-error-state">
      <div className="cm-error-icon">
        <AlertCircle size={48} />
      </div>
      <h3 className="cm-error-title">Error</h3>
      <p className="cm-error-message">{message}</p>
      {onRetry && (
        <button className="cm-error-button" onClick={onRetry}>
          {retryLabel}
        </button>
      )}
    </div>
  );
}

/**
 * EJEMPLO DE USO:
 * 
 * import ErrorState from './common/ErrorState';
 * 
 * const handleRetry = () => {
 *   console.log('Reintentando...');
 *   fetchData();
 * };
 * 
 * <ErrorState 
 *   message="No se pudieron cargar las estadísticas del servidor"
 *   onRetry={handleRetry}
 *   retryLabel="Volver a intentar"
 * />
 */


