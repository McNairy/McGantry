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
  inputs?: string;
  outputs?: string;
  triggeredBy: string;
  startedAt: string;
  completedAt?: string;
  error?: string;
}

export interface AuditEntry {
  id: string;
  timestamp: string;
  userId?: string;
  userName?: string;
  action: string;
  resourceType?: string;
  resourceId?: string;
  resourceName?: string;
  beforeState?: string;
  afterState?: string;
  source?: string;
  ipAddress?: string;
}

export interface APIKey {
  id: string;
  userId: string;
  name: string;
  prefix: string;
  role: string;
  createdAt: string;
  lastUsedAt?: string;
  expiresAt?: string;
  key?: string; // only present on creation response
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
  /** Custom extension: entity kind this field references (e.g. "API", "Team"). */
  'x-entity-ref'?: string;
}

export interface GraphNode {
  id: string;
  kind: string;
  namespace?: string;
  name: string;
  title?: string;
  isRoot: boolean;
}

export interface GraphEdge {
  from: string;
  to: string;
  relation: string;
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface PluginRegistryEntry {
  name: string;
  title: string;
  description: string;
  version: string;
  author: string;
  category: 'integration' | 'widget' | 'entity-kind' | 'action-type' | 'auth-provider';
  iconUrl?: string;
  homepage?: string;
  installed: boolean;
  enabled: boolean;
  installedAt?: string;
}

export interface PluginDetail {
  id: string;
  name: string;
  version: string;
  enabled: boolean;
  config?: Record<string, any>;
  manifest?: {
    name: string;
    title: string;
    description: string;
    version: string;
    author: string;
    category: string;
    configSchema?: Record<string, any>;
    entityPanels?: string[];
    actionTypes?: string[];
  };
  installedAt: string;
  updatedAt: string;
}

export interface PluginConfig {
  schema?: Record<string, any>;
  values?: Record<string, any>;
}

export interface PluginSyncResult {
  namespaces: number;
  deployments: number;
  services: number;
  created: number;
  updated: number;
  errors?: string[];
}

export interface K8sContainerInfo {
  name: string;
  image: string;
  ready: boolean;
  restarts: number;
  state: 'running' | 'waiting' | 'terminated' | 'unknown';
  reason?: string;
}

export interface K8sPodInfo {
  name: string;
  namespace: string;
  phase: string;
  ready: boolean;
  totalRestarts: number;
  nodeName?: string;
  startTime?: string;
  clusterName?: string;
  containers: K8sContainerInfo[];
}

export interface K8sDeploymentInfo {
  name: string;
  namespace: string;
  desiredReplicas: number;
  readyReplicas: number;
  availableReplicas: number;
}

export interface K8sWorkloadInfo {
  appName: string;
  deployments: K8sDeploymentInfo[];
  pods: K8sPodInfo[];
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
