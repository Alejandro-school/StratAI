import React from 'react';
import { createRoot } from 'react-dom/client';
import LandingWorld from './index';

const root = import.meta.hot?.data.root || createRoot(document.getElementById('root'));
root.render(<LandingWorld />);
if (import.meta.hot) import.meta.hot.data.root = root;
