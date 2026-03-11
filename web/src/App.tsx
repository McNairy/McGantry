import { useEffect } from 'react';
import { Routes, Route } from 'react-router-dom';
import { useAuth } from './hooks/useAuth';
import Sidebar from './components/Sidebar';
import CommandPalette from './components/CommandPalette';
import ErrorBoundary from './components/ErrorBoundary';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Catalog from './pages/Catalog';
import EntityDetail from './pages/EntityDetail';
import Actions from './pages/Actions';
import AuditLog from './pages/AuditLog';
import Settings from './pages/Settings';
import UsersPage from './pages/Users';
import Plugins from './pages/Plugins';
import StatusMonitor from './pages/StatusMonitor';
import GitOps from './pages/GitOps';

function AuthenticatedLayout() {
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <main className="flex-1 overflow-y-auto bg-[var(--gantry-bg-secondary)]">
        <div className="mx-auto max-w-7xl px-6 py-8">
          <ErrorBoundary>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/catalog" element={<Catalog />} />
              <Route path="/catalog/:kind" element={<Catalog />} />
              <Route path="/catalog/:kind/:name" element={<EntityDetail />} />
              <Route path="/actions" element={<Actions />} />
              <Route path="/audit" element={<AuditLog />} />
              <Route path="/users" element={<UsersPage />} />
              <Route path="/plugins" element={<Plugins />} />
              <Route path="/status" element={<StatusMonitor />} />
              <Route path="/gitops" element={<GitOps />} />
              <Route path="/settings" element={<Settings />} />
            </Routes>
          </ErrorBoundary>
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
  const { isAuthenticated, loading, loginWithToken } = useAuth();

  // Handle GitHub OAuth callback: server redirects to /?github_token=<jwt>
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const githubToken = params.get('github_token');
    if (githubToken) {
      // Clean the token from the URL before doing anything else.
      window.history.replaceState({}, '', window.location.pathname);
      loginWithToken(githubToken);
    }
  }, [loginWithToken]);

  if (loading) return <LoadingScreen />;
  if (!isAuthenticated) return <Login />;
  return <AuthenticatedLayout />;
}
