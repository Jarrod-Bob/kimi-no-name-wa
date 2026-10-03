import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
// Latin subsets only: the kanji the UI prints come from kanji-fonts.css,
// and anything else falls back to the system's Japanese fonts.
import '@fontsource/shippori-mincho/latin-600.css';
import '@fontsource/shippori-mincho/latin-800.css';
import '@fontsource/zen-kaku-gothic-new/latin-400.css';
import '@fontsource/zen-kaku-gothic-new/latin-500.css';
import '@fontsource/zen-kaku-gothic-new/latin-700.css';
import '@fontsource/dm-mono/latin-400.css';
import './styles/kanji-fonts.css';
import './styles/index.css';
import App from './App.tsx';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
);
