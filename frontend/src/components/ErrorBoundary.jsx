// frontend/src/components/ErrorBoundary.jsx
import React from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';

/**
 * Generic error boundary that catches render errors in its children.
 * Shows a recoverable fallback UI instead of crashing the whole page.
 */
class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error };
  }

  componentDidCatch(error, errorInfo) {
    console.error(`[ErrorBoundary:${this.props.name || 'unknown'}]`, error, errorInfo);
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const isInline = this.props.inline;

      return (
        <div
          style={{
            display: 'flex',
            flexDirection: isInline ? 'row' : 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: isInline ? '8px' : '12px',
            padding: isInline ? '12px 16px' : '24px',
            background: 'rgba(239, 68, 68, 0.08)',
            border: '1px solid rgba(239, 68, 68, 0.2)',
            borderRadius: '12px',
            color: '#94a3b8',
            fontSize: '13px',
            minHeight: isInline ? undefined : '120px',
          }}
        >
          <AlertTriangle size={isInline ? 16 : 24} color="#ef4444" />
          <span>{this.props.message || 'Algo salió mal'}</span>
          <button
            onClick={this.handleRetry}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '4px',
              padding: '6px 12px',
              background: 'rgba(99, 102, 241, 0.2)',
              border: '1px solid rgba(99, 102, 241, 0.4)',
              borderRadius: '6px',
              color: '#818cf8',
              cursor: 'pointer',
              fontSize: '12px',
              fontWeight: 600,
            }}
          >
            <RefreshCw size={12} />
            Reintentar
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
