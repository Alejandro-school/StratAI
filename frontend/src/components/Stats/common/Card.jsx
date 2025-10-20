// frontend/src/components/Stats/common/Card.jsx
import React from "react";
import "../../../styles/Stats/common.css";

/**
 * Card - Componente contenedor reutilizable con estilo glass
 * 
 * @param {React.ReactNode} children - Contenido de la card
 * @param {string} className - Clases CSS adicionales opcionales
 * @param {Function} onClick - Manejador de click opcional
 * @param {boolean} hoverable - Si debe tener efecto hover
 */
export default function Card({ children, className = "", onClick, hoverable = false }) {
  const cardClasses = `cm-card ${hoverable ? "cm-card-hoverable" : ""} ${className}`.trim();
  
  return (
    <div className={cardClasses} onClick={onClick}>
      {children}
    </div>
  );
}

/**
 * EJEMPLO DE USO:
 * 
 * import Card from './common/Card';
 * 
 * <Card hoverable>
 *   <h3>Contenido de la tarjeta</h3>
 *   <p>Descripción o datos</p>
 * </Card>
 * 
 * <Card className="custom-class" onClick={handleClick}>
 *   <p>Card con clase personalizada</p>
 * </Card>
 */


