import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Box,
  Zap,
  Settings,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  ChevronUp,
  LogOut,
  Server,
  Globe,
  Database,
  Users,
  Cloud,
  FileText,
  ClipboardList,
} from 'lucide-react';
import { useAuth } from '../hooks/useAuth';
import ThemeToggle from './ThemeToggle';
import { ENTITY_KINDS } from '../lib/types';

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  Server,
  Globe,
  Database,
  Users,
  Cloud,
  FileText,
};

interface NavItem {
  label: string;
  path: string;
  icon: React.ComponentType<{ className?: string }>;
  children?: { label: string; path: string; icon: React.ComponentType<{ className?: string }> }[];
}

const navItems: NavItem[] = [
  { label: 'Dashboard', path: '/', icon: LayoutDashboard },
  {
    label: 'Catalog',
    path: '/catalog',
    icon: Box,
    children: ENTITY_KINDS.map((k) => ({
      label: k.name,
      path: `/catalog/${k.name}`,
      icon: iconMap[k.icon] || Box,
    })),
  },
  { label: 'Actions', path: '/actions', icon: Zap },
  { label: 'Audit Log', path: '/audit', icon: ClipboardList },
  { label: 'Settings', path: '/settings', icon: Settings },
];

export default function Sidebar() {
  const [collapsed, setCollapsed] = useState(false);
  const [catalogOpen, setCatalogOpen] = useState(false);
  const location = useLocation();
  const { user, logout } = useAuth();

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/';
    return location.pathname.startsWith(path);
  };

  return (
    <aside
      className={`flex flex-col border-r border-[var(--gantry-border)] bg-[var(--gantry-bg-primary)] transition-all duration-200 ${
        collapsed ? 'w-16' : 'w-64'
      }`}
    >
      {/* Brand */}
      <div className="flex h-16 items-center justify-between border-b border-[var(--gantry-border)] px-4">
        {!collapsed && (
          <Link to="/" className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[var(--gantry-accent)] text-sm font-bold text-[var(--gantry-bg-primary)]">
              G
            </div>
            <span className="text-lg font-semibold text-[var(--gantry-text-primary)]">
              Gantry
            </span>
          </Link>
        )}
        {collapsed && (
          <Link to="/" className="mx-auto">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[var(--gantry-accent)] text-sm font-bold text-[var(--gantry-bg-primary)]">
              G
            </div>
          </Link>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <ul className="flex flex-col gap-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = isActive(item.path);
            const hasChildren = item.children && item.children.length > 0;

            return (
              <li key={item.path}>
                <div className="flex items-center">
                  <Link
                    to={item.path}
                    className={`flex flex-1 items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                      active
                        ? 'bg-[var(--gantry-accent)]/10 text-[var(--gantry-accent)]'
                        : 'text-[var(--gantry-text-secondary)] hover:bg-[var(--gantry-bg-tertiary)] hover:text-[var(--gantry-text-primary)]'
                    }`}
                    title={collapsed ? item.label : undefined}
                  >
                    <Icon className="h-5 w-5 shrink-0" />
                    {!collapsed && <span className="truncate">{item.label}</span>}
                  </Link>
                  {hasChildren && !collapsed && (
                    <button
                      onClick={() => setCatalogOpen(!catalogOpen)}
                      className="rounded p-1 text-[var(--gantry-text-secondary)] hover:bg-[var(--gantry-bg-tertiary)]"
                    >
                      {catalogOpen ? (
                        <ChevronUp className="h-4 w-4" />
                      ) : (
                        <ChevronDown className="h-4 w-4" />
                      )}
                    </button>
                  )}
                </div>
                {hasChildren && catalogOpen && !collapsed && (
                  <ul className="ml-4 mt-1 flex flex-col gap-0.5 border-l border-[var(--gantry-border)] pl-3">
                    {item.children!.map((child) => {
                      const ChildIcon = child.icon;
                      const childActive = isActive(child.path);
                      return (
                        <li key={child.path}>
                          <Link
                            to={child.path}
                            className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors ${
                              childActive
                                ? 'text-[var(--gantry-accent)]'
                                : 'text-[var(--gantry-text-secondary)] hover:text-[var(--gantry-text-primary)]'
                            }`}
                          >
                            <ChildIcon className="h-4 w-4 shrink-0" />
                            <span>{child.label}</span>
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                )}
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Bottom section */}
      <div className="border-t border-[var(--gantry-border)] p-3">
        <div className="flex items-center justify-between">
          {!collapsed && <ThemeToggle />}
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="rounded-lg p-2 text-[var(--gantry-text-secondary)] hover:bg-[var(--gantry-bg-tertiary)]"
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <ChevronLeft className="h-4 w-4" />
            )}
          </button>
        </div>
        {!collapsed && user && (
          <div className="mt-3 flex items-center justify-between rounded-lg bg-[var(--gantry-bg-secondary)] px-3 py-2">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-[var(--gantry-text-primary)]">
                {user.displayName || user.username}
              </p>
              <p className="truncate text-xs text-[var(--gantry-text-secondary)]">{user.role}</p>
            </div>
            <button
              onClick={logout}
              className="shrink-0 rounded p-1.5 text-[var(--gantry-text-secondary)] hover:bg-[var(--gantry-bg-tertiary)] hover:text-[var(--gantry-danger)]"
              title="Logout"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        )}
        {collapsed && user && (
          <button
            onClick={logout}
            className="mt-2 flex w-full items-center justify-center rounded-lg p-2 text-[var(--gantry-text-secondary)] hover:bg-[var(--gantry-bg-tertiary)] hover:text-[var(--gantry-danger)]"
            title="Logout"
          >
            <LogOut className="h-4 w-4" />
          </button>
        )}
      </div>
    </aside>
  );
}
