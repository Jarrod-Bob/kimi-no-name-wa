import { Route, Routes } from 'react-router-dom';
import { Shell } from './components/Shell';
import { FavouritesPage } from './pages/FavouritesPage';
import { GeneratePage } from './pages/GeneratePage';
import { HistoryPage } from './pages/HistoryPage';
import { SettingsPage } from './pages/SettingsPage';
import { AppStateProvider } from './session/AppState';

export default function App() {
  return (
    <AppStateProvider>
      <Routes>
        <Route element={<Shell />}>
          <Route index element={<GeneratePage />} />
          <Route path="favourites" element={<FavouritesPage />} />
          <Route path="history" element={<HistoryPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="*" element={<GeneratePage />} />
        </Route>
      </Routes>
    </AppStateProvider>
  );
}
