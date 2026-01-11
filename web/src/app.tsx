import { Router, Route } from 'preact-router';
import type { JSX } from 'preact';
import { Header } from './components/Header';
import { Overview } from './pages/Overview';
import { TodaysDetections } from './pages/TodaysDetections';
import { History } from './pages/History';
import { Stats } from './pages/Stats';
import { SpeciesManagement } from './pages/SpeciesManagement';
import { Spectrogram } from './pages/Spectrogram';

/**
 * Main application component with routing.
 */
export function App(): JSX.Element {
  return (
    <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
      <Header />
      <main class="container mx-auto px-4 py-6">
        <Router>
          <Route path="/app/" component={Overview} />
          <Route path="/app/detections" component={TodaysDetections} />
          <Route path="/app/history" component={History} />
          <Route path="/app/stats" component={Stats} />
          <Route path="/app/species" component={SpeciesManagement} />
          <Route path="/app/live" component={Spectrogram} />
          <Route default component={Overview} />
        </Router>
      </main>
    </div>
  );
}
