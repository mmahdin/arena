import React from 'react';
import { createRoot } from 'react-dom/client';
import '@fontsource-variable/estedad';
import '@fontsource-variable/vazirmatn';
import '@fontsource-variable/noto-naskh-arabic';
import App from './App';
import './styles.css';

createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
