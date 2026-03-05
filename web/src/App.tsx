import { Routes, Route } from 'react-router-dom';
import { useAuth } from './hooks/useAuth';
import Sidebar from './components/Sidebar';
import CommandPalette from './components/CommandPalette';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Catalog from './pages/Catalog';
import EntityDetail from './pages/EntityDetail';
import Actions from './pages/Actions';
import Settings from './pages/Settings';

function AuthenticatedLayout() {
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <main className="flex-1 overflow-y-auto bg-[var(--gantry-bg-secondary)]">
        <div className="mx-auto max-w-7xl px-6 py-8">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/catalog" element={<Catalog />} />
            <Route path="/catalog/:kind" element={<Catalog />} />
            <Route path="/catalog/:kind/:name" element={<EntityDetail />} />
            <Route path="/actions" element={<Actions />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </div>
      </main>
      <CommandPalette />
    </div>
  );
}

function LoadingScreen() {
  return (
    <div className="flex h-screen items-center justify-center bg-[var(--gantry-bg-primary)]">
      <div className="flex flex-col items-center gap-4">
        <div className="spinner h-8 w-8 text-[var(--gantry-accent)]" />
        <p className="text-sm text-[var(--gantry-text-secondary)]">Loading Gantry...</p>
      </div>
    </div>
  );
}

export default function App() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) return <LoadingScreen />;
  if (!isAuthenticated) return <Login />;
  return <AuthenticatedLayout />;
}
