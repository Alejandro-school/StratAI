// Button - Reusable button component (primary/secondary/skip)
import React from 'react';
import '../../../styles/Landing/shared.css';

/**
 * Button Component
 * 
 * Variants:
 * - primary: Dominant button with electric gradient and glow (for main CTAs)
 * - secondary: Subtle button with border (for secondary actions)
 * - ghost: Minimal text button (for tertiary actions)
 * 
 * @param {string} variant - 'primary' | 'secondary' | 'ghost'
 * @param {string} size - 'sm' | 'md' | 'lg' | 'xl'
 * @param {boolean} pulse - Enable pulse animation (primary only)
 * @param {ReactNode} icon - Optional icon element
 * @param {string} iconPosition - 'left' | 'right'
 */
const Button = ({
  children,
  variant = 'primary',
  size = 'md',
  pulse = false,
  icon = null,
  iconPosition = 'right',
  className = '',
  disabled = false,
  onClick,
  ...props
}) => {
  const classes = [
    'landing-btn',
    `landing-btn--${variant}`,
    size !== 'md' && `landing-btn--${size}`,
    pulse && variant === 'primary' && 'landing-btn--pulse',
    className,
  ].filter(Boolean).join(' ');

  const iconElement = icon && (
    <span className={`landing-btn__icon landing-btn__icon--${iconPosition === 'right' ? 'arrow' : 'left'}`}>
      {icon}
    </span>
  );

  return (
    <button
      className={classes}
      disabled={disabled}
      onClick={onClick}
      {...props}
    >
      {iconPosition === 'left' && iconElement}
      {children}
      {iconPosition === 'right' && iconElement}
    </button>
  );
};

// ═══════════════════════════════════════════════════════════════════════════
// Common Icons
// ═══════════════════════════════════════════════════════════════════════════
export const ArrowRightIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M5 12h14M12 5l7 7-7 7" />
  </svg>
);

export const ChevronRightIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M9 18l6-6-6-6" />
  </svg>
);

export const SkipIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <polygon points="5 4 15 12 5 20 5 4" />
    <line x1="19" y1="5" x2="19" y2="19" />
  </svg>
);

export default Button;
