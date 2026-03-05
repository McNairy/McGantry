import { useAuth } from '../hooks/useAuth';
import { useTheme } from '../hooks/useTheme';
import { Sun, Moon, User, Shield, Info } from 'lucide-react';

export default function Settings() {
  const { user, logout } = useAuth();
  const { theme, toggle } = useTheme();

  return (
    <div>
      <h1 className="text-2xl font-bold text-[var(--gantry-text-primary)]">Settings</h1>
      <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">Manage your preferences</p>

      <div className="mt-8 space-y-6">
        {/* Profile */}
        <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-[var(--gantry-accent)]/10 text-[var(--gantry-accent)]">
              <User className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Profile</h2>
              <p className="text-xs text-[var(--gantry-text-secondary)]">Your account information</p>
            </div>
          </div>
          <dl className="mt-4 space-y-3">
            <div className="flex items-center justify-between">
              <dt className="text-sm text-[var(--gantry-text-secondary)]">Username</dt>
              <dd className="text-sm font-medium text-[var(--gantry-text-primary)]">{user?.username || '-'}</dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="text-sm text-[var(--gantry-text-secondary)]">Email</dt>
              <dd className="text-sm font-medium text-[var(--gantry-text-primary)]">{user?.email || '-'}</dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="text-sm text-[var(--gantry-text-secondary)]">Role</dt>
              <dd className="flex items-center gap-1.5 text-sm font-medium text-[var(--gantry-text-primary)]">
                <Shield className="h-3.5 w-3.5 text-[var(--gantry-accent)]" />
                {user?.role || '-'}
              </dd>
            </div>
          </dl>
          <button
            onClick={logout}
            className="mt-4 rounded-lg border border-red-200 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20"
          >
            Sign Out
          </button>
        </div>

        {/* Appearance */}
        <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
          <h2 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Appearance</h2>
          <div className="mt-4 flex items-center justify-between">
            <div className="flex items-center gap-3">
              {theme === 'dark' ? <Moon className="h-5 w-5 text-[var(--gantry-text-secondary)]" /> : <Sun className="h-5 w-5 text-[var(--gantry-text-secondary)]" />}
              <div>
                <p className="text-sm text-[var(--gantry-text-primary)]">Theme</p>
                <p className="text-xs text-[var(--gantry-text-secondary)]">
                  Currently using {theme} mode
                </p>
              </div>
            </div>
            <button
              onClick={toggle}
              className={`relative h-6 w-11 rounded-full transition-colors ${
                theme === 'dark' ? 'bg-[var(--gantry-accent)]' : 'bg-[var(--gantry-bg-tertiary)]'
              }`}
            >
              <span
                className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${
                  theme === 'dark' ? 'translate-x-5' : 'translate-x-0'
                }`}
              />
            </button>
          </div>
        </div>

        {/* About */}
        <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-6">
          <div className="flex items-center gap-3">
            <Info className="h-5 w-5 text-[var(--gantry-text-secondary)]" />
            <h2 className="text-sm font-semibold text-[var(--gantry-text-primary)]">About</h2>
          </div>
          <dl className="mt-4 space-y-2">
            <div className="flex items-center justify-between">
              <dt className="text-sm text-[var(--gantry-text-secondary)]">Version</dt>
              <dd className="text-sm font-medium text-[var(--gantry-text-primary)]">0.1.0</dd>
            </div>
            <div className="flex items-center justify-between">
              <dt className="text-sm text-[var(--gantry-text-secondary)]">License</dt>
              <dd className="text-sm font-medium text-[var(--gantry-text-primary)]">Apache 2.0</dd>
            </div>
          </dl>
          <p className="mt-4 text-xs text-[var(--gantry-text-secondary)]">
            Gantry — The Developer Platform That Just Works
          </p>
        </div>
      </div>
    </div>
  );
}
