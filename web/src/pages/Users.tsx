import { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { Link } from 'react-router-dom';
import { Plus, Trash2, Shield, Users, Info } from 'lucide-react';
import { api } from '../lib/api';
import type { User } from '../lib/types';

const ROLE_COLORS: Record<string, string> = {
  admin: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  'platform-engineer': 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  developer: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  viewer: 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400',
};

export default function UsersPage() {
  const { user: me } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Create form
  const [newUsername, setNewUsername] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newDisplayName, setNewDisplayName] = useState('');
  const [newEmail, setNewEmail] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');

  useEffect(() => {
    api.listUsers()
      .then((list) => setUsers(list ?? []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const handleCreate = async () => {
    setCreateError('');
    if (!newUsername.trim() || !newPassword.trim()) return;
    setCreating(true);
    try {
      const created = await api.createUser(
        newUsername.trim(),
        newPassword.trim(),
        newDisplayName.trim() || undefined,
        newEmail.trim() || undefined,
      );
      setUsers((prev) => [...prev, created]);
      setNewUsername('');
      setNewPassword('');
      setNewDisplayName('');
      setNewEmail('');
    } catch (e: any) {
      setCreateError(e.message);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await api.deleteUser(id);
      setUsers((prev) => prev.filter((u) => u.id !== id));
    } catch {
      // silently ignore
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--gantry-text-primary)]">Users</h1>
          <p className="mt-1 text-sm text-[var(--gantry-text-secondary)]">
            Manage user accounts. Roles are assigned via{' '}
            <Link to="/rbac" className="text-[var(--gantry-accent)] hover:underline">
              Access Control
            </Link>{' '}
            groups.
          </p>
        </div>
        <div className="flex items-center gap-2 rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] px-3 py-2">
          <Users className="h-4 w-4 text-[var(--gantry-text-secondary)]" />
          <span className="text-sm font-medium text-[var(--gantry-text-primary)]">{users.length}</span>
          <span className="text-sm text-[var(--gantry-text-secondary)]">
            {users.length === 1 ? 'user' : 'users'}
          </span>
        </div>
      </div>

      {error && (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {error}
        </div>
      )}

      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Create user panel */}
        <div className="lg:col-span-1">
          <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-5">
            <h2 className="text-sm font-semibold text-[var(--gantry-text-primary)]">Add User</h2>
            <p className="mt-0.5 text-xs text-[var(--gantry-text-secondary)]">Create a new account</p>

            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-xs font-medium text-[var(--gantry-text-secondary)]">
                  Username <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={newUsername}
                  onChange={(e) => setNewUsername(e.target.value)}
                  placeholder="e.g. jsmith"
                  className="mt-1 w-full rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-3 py-2 text-sm text-[var(--gantry-text-primary)] placeholder-[var(--gantry-text-secondary)] outline-none focus:border-[var(--gantry-accent)]"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-[var(--gantry-text-secondary)]">
                  Password <span className="text-red-500">*</span>
                </label>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="Min. 8 characters"
                  className="mt-1 w-full rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-3 py-2 text-sm text-[var(--gantry-text-primary)] placeholder-[var(--gantry-text-secondary)] outline-none focus:border-[var(--gantry-accent)]"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-[var(--gantry-text-secondary)]">Display Name</label>
                <input
                  type="text"
                  value={newDisplayName}
                  onChange={(e) => setNewDisplayName(e.target.value)}
                  placeholder="e.g. Jane Smith"
                  className="mt-1 w-full rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-3 py-2 text-sm text-[var(--gantry-text-primary)] placeholder-[var(--gantry-text-secondary)] outline-none focus:border-[var(--gantry-accent)]"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-[var(--gantry-text-secondary)]">Email</label>
                <input
                  type="email"
                  value={newEmail}
                  onChange={(e) => setNewEmail(e.target.value)}
                  placeholder="jane@example.com"
                  className="mt-1 w-full rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-secondary)] px-3 py-2 text-sm text-[var(--gantry-text-primary)] placeholder-[var(--gantry-text-secondary)] outline-none focus:border-[var(--gantry-accent)]"
                />
              </div>
            </div>

            {createError && (
              <p className="mt-3 text-xs text-red-500">{createError}</p>
            )}

            <button
              onClick={handleCreate}
              disabled={creating || !newUsername.trim() || !newPassword.trim()}
              className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-[var(--gantry-accent)] px-4 py-2 text-sm font-medium text-[var(--gantry-bg-primary)] hover:bg-[var(--gantry-accent-hover)] disabled:opacity-50"
            >
              <Plus className="h-4 w-4" />
              {creating ? 'Creating…' : 'Create User'}
            </button>
          </div>

          {/* SSO hint */}
          <div className="mt-4 rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-5">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-[var(--gantry-text-primary)]">
              <Info className="h-4 w-4 text-[var(--gantry-text-secondary)]" />
              SSO Users
            </h3>
            <p className="mt-2 text-xs text-[var(--gantry-text-secondary)]">
              For users who will sign in via GitHub SSO, create their account with the username{' '}
              <code className="rounded bg-[var(--gantry-bg-tertiary)] px-1 py-0.5 text-[var(--gantry-text-primary)]">
                github:&lt;username&gt;
              </code>{' '}
              or use the same email address as their GitHub account. The password can be any value — SSO users authenticate through GitHub.
            </p>
          </div>

          {/* Manage roles hint */}
          <div className="mt-4 rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] p-5">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-[var(--gantry-text-primary)]">
              <Shield className="h-4 w-4 text-[var(--gantry-text-secondary)]" />
              Managing Roles
            </h3>
            <p className="mt-2 text-xs text-[var(--gantry-text-secondary)]">
              New users start as <strong>viewer</strong>. To grant higher access, add them to a group in{' '}
              <Link to="/rbac" className="text-[var(--gantry-accent)] hover:underline">
                Access Control
              </Link>
              . Default groups: Admins, Platform Engineers, Developers.
            </p>
          </div>
        </div>

        {/* User list */}
        <div className="lg:col-span-2">
          <div className="rounded-lg border border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)]">
            {loading ? (
              <div className="flex items-center justify-center py-16">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--gantry-accent)] border-t-transparent" />
              </div>
            ) : users.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-16">
                <Users className="h-8 w-8 text-[var(--gantry-text-secondary)]" />
                <p className="mt-2 text-sm text-[var(--gantry-text-secondary)]">No users yet</p>
              </div>
            ) : (
              <table className="w-full">
                <thead>
                  <tr className="border-b border-[var(--gantry-border)]">
                    <th className="px-4 py-3 text-left text-xs font-medium text-[var(--gantry-text-secondary)]">User</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-[var(--gantry-text-secondary)]">Email</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-[var(--gantry-text-secondary)]">Groups</th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-[var(--gantry-text-secondary)]">Effective Role</th>
                    <th className="px-4 py-3" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--gantry-border)]">
                  {users.map((u) => (
                    <tr key={u.id} className="group">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[var(--gantry-bg-tertiary)] text-xs font-semibold text-[var(--gantry-text-primary)]">
                            {(u.displayName || u.username)[0].toUpperCase()}
                          </div>
                          <div>
                            <p className="text-sm font-medium text-[var(--gantry-text-primary)]">
                              {u.displayName || u.username}
                              {u.id === me?.id && (
                                <span className="ml-1.5 text-xs text-[var(--gantry-text-secondary)]">(you)</span>
                              )}
                            </p>
                            <p className="text-xs text-[var(--gantry-text-secondary)]">@{u.username}</p>
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm text-[var(--gantry-text-secondary)]">
                        {u.email || <span className="text-[var(--gantry-text-secondary)]/40">—</span>}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap gap-1">
                          {u.groups && u.groups.length > 0 ? (
                            u.groups.map((g) => (
                              <span
                                key={g}
                                className="inline-flex rounded-full bg-[var(--gantry-accent)]/10 px-2 py-0.5 text-xs font-medium text-[var(--gantry-accent)]"
                              >
                                {g}
                              </span>
                            ))
                          ) : (
                            <span className="text-xs text-[var(--gantry-text-secondary)]/40">—</span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${ROLE_COLORS[u.effectiveRole || u.role] || ROLE_COLORS.viewer}`}>
                          {u.effectiveRole || u.role}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => handleDelete(u.id)}
                          disabled={u.id === me?.id}
                          className="rounded p-1.5 text-[var(--gantry-text-secondary)] opacity-0 transition-opacity group-hover:opacity-100 hover:bg-[var(--gantry-bg-tertiary)] hover:text-[var(--gantry-danger)] disabled:pointer-events-none disabled:opacity-0"
                          title="Delete user"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
