import { Router, Route } from 'preact-router';
import type { JSX } from 'preact';
import { Header } from './components/Header';
import { Overview } from './pages/Overview';
import { Detections } from './pages/Detections';
import { Stats } from './pages/Stats';
import { SpeciesManagement } from './pages/SpeciesManagement';
import { Spectrogram } from './pages/Spectrogram';
import { Recordings } from './pages/Recordings';
import { Settings } from './pages/Settings';
import { AdvancedSettings } from './pages/AdvancedSettings';
import { ServiceControls } from './components/ServiceControls';
import { Backup } from './pages/Backup';

/**
 * Main application component with routing.
 */
export function App(): JSX.Element {
  return (
    <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
      <Header />
      <main class="container mx-auto px-4 py-6">
        <Router>
          <Route path="/" component={Overview} />
          <Route path="/detections" component={Detections} />
          <Route path="/stats" component={Stats} />
          <Route path="/species" component={SpeciesManagement} />
          <Route path="/live" component={Spectrogram} />
          <Route path="/recordings" component={Recordings} />
          <Route path="/settings" component={Settings} />
          <Route path="/advanced-settings" component={AdvancedSettings} />
          <Route path="/services" component={ServiceControls} />
          <Route path="/backup" component={Backup} />
          <Route default component={Overview} />
        </Router>
      </main>
    </div>
  );
}
