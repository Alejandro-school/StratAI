import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { CssBaseline } from '@mui/material';
import { ThemeProvider, createTheme, StyledEngineProvider } from '@mui/material/styles';
import './styles/responsive.css'; // Global responsive styles
import './styles/theme/landingPaletteGlobal.css';

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#06b6d4' },
    secondary: { main: '#8b5cf6' },
    success: { main: '#10b981' },
    warning: { main: '#f59e0b' },
    error: { main: '#ef4444' },
    background: {
      default: '#0c1219',
      paper: '#10161f'
    },
    text: {
      primary: '#ffffff',
      secondary: 'rgba(255, 255, 255, 0.72)'
    }
  },
  typography: { fontFamily: 'Inter, sans-serif' }
});

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <StyledEngineProvider injectFirst>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <App />
      </ThemeProvider>
    </StyledEngineProvider>
  </React.StrictMode>
);