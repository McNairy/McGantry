export interface Entity {
  kind: string;
  apiVersion: string;
  metadata: EntityMetadata;
  spec?: Record<string, any>;
}

export interface EntityMetadata {
  name: string;
  namespace?: string;
  title?: string;
  description?: string;
  owner?: string;
  tags?: string[];
  annotations?: Record<string, string>;
  labels?: Record<string, string>;
  createdAt?: string;
  updatedAt?: string;
  createdBy?: string;
}

export interface User {
  id: string;
  username: string;
  displayName?: string;
  email?: string;
  role: string;
}

export interface SearchResult {
  kind: string;
  name: string;
  namespace: string;
  title: string;
  rank: number;
}

export interface ActionDef {
  kind: string;
  apiVersion: string;
  metadata: EntityMetadata;
  spec?: {
    type?: string;
    description?: string;
    inputs?: Record<string, any>;
    steps?: any[];
  };
}

export interface ActionRun {
  id: string;
  actionName: string;
  status: string;
  inputs?: Record<string, any>;
  outputs?: Record<string, any>;
  triggeredBy: string;
  startedAt: string;
  completedAt?: string;
  error?: string;
}

export interface JsonSchema {
  type?: string;
  properties?: Record<string, JsonSchema>;
  required?: string[];
  items?: JsonSchema;
  enum?: any[];
  description?: string;
  title?: string;
  default?: any;
  format?: string;
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
}

export const ENTITY_KINDS = [
  { name: 'Service', plural: 'services', icon: 'Server' },
  { name: 'API', plural: 'apis', icon: 'Globe' },
  { name: 'Infrastructure', plural: 'infrastructure', icon: 'Database' },
  { name: 'Team', plural: 'teams', icon: 'Users' },
  { name: 'Environment', plural: 'environments', icon: 'Cloud' },
  { name: 'Documentation', plural: 'documentation', icon: 'FileText' },
] as const;

export type EntityKindName = (typeof ENTITY_KINDS)[number]['name'];
