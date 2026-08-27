import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './styles/responsive.css'; // Global responsive styles
import './styles/theme/landingPaletteGlobal.css';
import './styles/theme/globalBaseline.css';
import './styles/Layout/navigationFrame.css';

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
