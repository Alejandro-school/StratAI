import React, { useEffect, useRef } from 'react';
import { X } from 'lucide-react';

const ProgressDialog = ({ ariaLabel, children, eyebrow, title, onClose, size = 'default' }) => {
  const closeButtonRef = useRef(null);

  useEffect(() => {
    closeButtonRef.current?.focus();
    const handleKeyDown = (event) => {
      if (event.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  return (
    <div className="op-dialog-layer" role="presentation" onMouseDown={onClose}>
      <section
        className={`op-dialog op-dialog--${size}`}
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="op-dialog__header">
          <div>
            {eyebrow ? <span className="op-eyebrow">{eyebrow}</span> : null}
            <h2>{title}</h2>
          </div>
          <button ref={closeButtonRef} type="button" className="op-icon-button" onClick={onClose} aria-label="Cerrar">
            <X size={18} aria-hidden="true" />
          </button>
        </header>
        <div className="op-dialog__body">{children}</div>
      </section>
    </div>
  );
};

export default ProgressDialog;
