import { BrowserRouter } from 'react-router-dom';
import { AppShell } from './AppShell';
import { SettingsProvider } from '@/store/settings';
import { AppProvider } from '@/store/app-store';

export function App() {
  return (
    <SettingsProvider>
      <AppProvider>
        <BrowserRouter>
          <AppShell />
        </BrowserRouter>
      </AppProvider>
    </SettingsProvider>
  );
}
