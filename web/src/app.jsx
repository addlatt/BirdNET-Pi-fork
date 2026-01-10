import { Router, Route } from 'preact-router';
import { Header } from './components/Header';
import { Overview } from './pages/Overview';
import { TodaysDetections } from './pages/TodaysDetections';
import { Stats } from './pages/Stats';

export function App() {
  return (
    <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
      <Header />
      <main class="container mx-auto px-4 py-6">
        <Router>
          <Route path="/app/" component={Overview} />
          <Route path="/app/detections" component={TodaysDetections} />
          <Route path="/app/stats" component={Stats} />
          <Route default component={Overview} />
        </Router>
      </main>
    </div>
  );
}
