import type { Entity, User, SearchResult, ActionRun } from './types';

let authToken: string | null = localStorage.getItem('gantry_token');

function headers(): Record<string, string> {
  const h: Record<string, string> = { 'Content-Type': 'application/json' };
  if (authToken) h['Authorization'] = `Bearer ${authToken}`;
  return h;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    method,
    headers: headers(),
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export function setToken(token: string | null) {
  authToken = token;
  if (token) {
    localStorage.setItem('gantry_token', token);
  } else {
    localStorage.removeItem('gantry_token');
  }
}

export function getToken(): string | null {
  return authToken;
}

export const api = {
  login: (username: string, password: string) =>
    request<{ token: string; user: User }>('POST', '/auth/login', { username, password }),

  getMe: () => request<User>('GET', '/auth/me'),

  listEntities: (kind?: string) =>
    request<Entity[]>('GET', kind ? `/entities/${kind}` : '/entities'),

  getEntity: (kind: string, name: string) =>
    request<Entity>('GET', `/entities/${kind}/${name}`),

  createEntity: (entity: Entity) =>
    request<Entity>('POST', '/entities', entity),

  updateEntity: (kind: string, name: string, entity: Entity) =>
    request<Entity>('PUT', `/entities/${kind}/${name}`, entity),

  deleteEntity: (kind: string, name: string) =>
    request<void>('DELETE', `/entities/${kind}/${name}`),

  search: (q: string) =>
    request<SearchResult[]>('GET', `/search?q=${encodeURIComponent(q)}`),

  listSchemas: () => request<Record<string, any>>('GET', '/schemas'),

  getSchema: (kind: string) => request<any>('GET', `/schemas/${kind}`),

  listActions: () => request<Entity[]>('GET', '/actions'),

  executeAction: (name: string, inputs: Record<string, any>) =>
    request<ActionRun>('POST', `/actions/${name}/execute`, { inputs }),
};
