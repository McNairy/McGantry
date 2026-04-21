import type { CSSProperties } from 'react';
import type { FlowEdge, FlowEntityNode, FlowMockNode, FlowMockShape, FlowNode, FlowSpec } from './types';

export const ENTITY_NODE_WIDTH = 208;
export const ENTITY_NODE_HEIGHT = 92;
export const MAX_NODE_WIDTH = 440;
export const MIN_NODE_HEIGHT = 92;
export const MOCK_SHAPE_OPTIONS: FlowMockShape[] = ['box', 'pill', 'diamond', 'note'];

export const FLOW_KIND_COLORS: Record<string, string> = {
  Service: '#3B82F6',
  API: '#10B981',
  Infrastructure: '#F59E0B',
  Team: '#8B5CF6',
  Environment: '#06B6D4',
  Documentation: '#EC4899',
  Action: '#EF4444',
  Flow: '#0EA5E9',
};

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

export function ensureFlowSpec(spec: Record<string, any> | undefined): FlowSpec {
  const viewport = spec?.viewport || {};
  const nodes = Array.isArray(spec?.nodes) ? spec.nodes : [];
  const edges = Array.isArray(spec?.edges) ? spec.edges : [];

  return {
    viewport: {
      x: typeof viewport.x === 'number' ? viewport.x : 0,
      y: typeof viewport.y === 'number' ? viewport.y : 0,
      zoom: typeof viewport.zoom === 'number' ? viewport.zoom : 1,
    },
    nodes: nodes
      .filter((node: any) => node && typeof node === 'object')
      .map((node: any): FlowNode | null => {
        const position = {
          x: typeof node.position?.x === 'number' ? node.position.x : 0,
          y: typeof node.position?.y === 'number' ? node.position.y : 0,
        };

        if (node.nodeType === 'mock') {
          return {
            id: typeof node.id === 'string' ? node.id : crypto.randomUUID(),
            nodeType: 'mock',
            label: typeof node.label === 'string' && node.label.trim() ? node.label : 'Mock Node',
            subtitle: typeof node.subtitle === 'string' ? node.subtitle : '',
            shape: MOCK_SHAPE_OPTIONS.includes(node.shape) ? node.shape : 'box',
            color: typeof node.color === 'string' && node.color.trim() ? node.color : '#64748B',
            width: typeof node.width === 'number' ? node.width : undefined,
            height: typeof node.height === 'number' ? node.height : undefined,
            position,
            locked: typeof node.locked === 'boolean' ? node.locked : undefined,
            zIndex: typeof node.zIndex === 'number' ? node.zIndex : undefined,
            parentId: typeof node.parentId === 'string' ? node.parentId : undefined,
            badge: typeof node.badge === 'string' ? node.badge : undefined,
          };
        }

        const entityNode: FlowEntityNode = {
          id: typeof node.id === 'string' ? node.id : crypto.randomUUID(),
          nodeType: 'entity',
          entityRef: {
            kind: String(node.entityRef?.kind || ''),
            name: String(node.entityRef?.name || ''),
            namespace: typeof node.entityRef?.namespace === 'string' ? node.entityRef.namespace : undefined,
          },
          position,
          locked: typeof node.locked === 'boolean' ? node.locked : undefined,
          zIndex: typeof node.zIndex === 'number' ? node.zIndex : undefined,
          parentId: typeof node.parentId === 'string' ? node.parentId : undefined,
          badge: typeof node.badge === 'string' ? node.badge : undefined,
        };

        return entityNode.entityRef.kind && entityNode.entityRef.name ? entityNode : null;
      })
      .filter((node): node is FlowNode => Boolean(node)),
    edges: edges
      .filter((edge: any) => edge && typeof edge === 'object')
      .map((edge: any): FlowEdge => ({
        id: typeof edge.id === 'string' ? edge.id : crypto.randomUUID(),
        source: String(edge.source || ''),
        target: String(edge.target || ''),
        relation: String(edge.relation || 'calls'),
        direction: edge.direction === 'two-way' ? 'two-way' : 'one-way',
        label: typeof edge.label === 'string' ? edge.label : '',
        animated: typeof edge.animated === 'boolean' ? edge.animated : true,
      }))
      .filter((edge) => Boolean(edge.source && edge.target && edge.relation)),
  };
}

export function isMockNode(node: FlowNode): node is FlowMockNode {
  return node.nodeType === 'mock';
}

export function isEntityNode(node: FlowNode): node is FlowEntityNode {
  return !isMockNode(node);
}

export function nodeColor(node: FlowNode): string {
  if (isMockNode(node)) return node.color || '#64748B';
  return FLOW_KIND_COLORS[node.entityRef.kind] || '#64748B';
}

export function nodeSubtitle(node: FlowNode): string {
  if (isMockNode(node)) return node.subtitle || '';
  return node.entityRef.name;
}

export function nodeBadge(node: FlowNode): string {
  if (node.badge !== undefined) return node.badge;
  if (isMockNode(node)) return `Mock ${mockShapeLabel(node.shape)}`;
  return node.entityRef.kind;
}

export function mockShapeLabel(shape: FlowMockShape): string {
  switch (shape) {
    case 'pill':
      return 'Pill';
    case 'diamond':
      return 'Diamond';
    case 'note':
      return 'Note';
    default:
      return 'Box';
  }
}

export function withAlpha(color: string, alpha: string): string {
  if (/^#[0-9a-fA-F]{6}$/.test(color)) {
    return `${color}${alpha}`;
  }
  return color;
}

export function mockFoldFill(color: string): string {
  return withAlpha(color, '22');
}

export function estimateWrappedLines(text: string, charsPerLine: number): number {
  if (!text.trim()) return 0;
  return text
    .split(/\r?\n/)
    .reduce((total, line) => total + Math.max(1, Math.ceil(line.trim().length / Math.max(1, charsPerLine))), 0);
}

export function getNodeDimensions(node: FlowNode): { width: number; height: number } {
  if (isEntityNode(node)) {
    return { width: ENTITY_NODE_WIDTH, height: ENTITY_NODE_HEIGHT };
  }

  const title = node.label.trim();
  const subtitle = (node.subtitle || '').trim();
  const longestText = Math.max(title.length, subtitle.length, 12);

  switch (node.shape) {
    case 'pill': {
      const baseWidth = clamp(188 + Math.max(0, longestText - 12) * 7, 188, 340);
      const width = clamp(Math.max(node.width || 0, baseWidth), 188, MAX_NODE_WIDTH);
      const charsPerLine = Math.max(12, Math.floor((width - 44) / 8));
      const titleLines = estimateWrappedLines(title, charsPerLine);
      const subtitleLines = estimateWrappedLines(subtitle, charsPerLine);
      const baseHeight = Math.max(84, 48 + titleLines * 18 + subtitleLines * 16 + 18);
      const height = Math.max(node.height || 0, baseHeight);
      return { width, height };
    }
    case 'diamond': {
      const baseWidth = clamp(272 + Math.max(0, longestText - 10) * 10, 272, MAX_NODE_WIDTH);
      const width = clamp(Math.max(node.width || 0, baseWidth), 272, MAX_NODE_WIDTH);
      const charsPerLine = Math.max(9, Math.floor((width * 0.42) / 8));
      const titleLines = estimateWrappedLines(title, charsPerLine);
      const subtitleLines = estimateWrappedLines(subtitle, charsPerLine);
      const baseHeight = clamp(120 + Math.max(0, titleLines - 1) * 22 + subtitleLines * 20, 120, 220);
      const height = Math.max(node.height || 0, baseHeight);
      return { width, height };
    }
    case 'note': {
      const baseWidth = clamp(196 + Math.max(0, longestText - 14) * 7, 196, 360);
      const width = clamp(Math.max(node.width || 0, baseWidth), 196, MAX_NODE_WIDTH);
      const charsPerLine = Math.max(13, Math.floor((width - 52) / 8));
      const titleLines = estimateWrappedLines(title, charsPerLine);
      const subtitleLines = estimateWrappedLines(subtitle, charsPerLine);
      const baseHeight = Math.max(MIN_NODE_HEIGHT, 56 + titleLines * 18 + subtitleLines * 16 + 24);
      const height = Math.max(node.height || 0, baseHeight);
      return { width, height };
    }
    case 'box':
    default: {
      const baseWidth = clamp(180 + Math.max(0, longestText - 14) * 6, 180, 320);
      const width = clamp(Math.max(node.width || 0, baseWidth), 180, MAX_NODE_WIDTH);
      const charsPerLine = Math.max(14, Math.floor((width - 36) / 8));
      const titleLines = estimateWrappedLines(title, charsPerLine);
      const subtitleLines = estimateWrappedLines(subtitle, charsPerLine);
      const baseHeight = Math.max(MIN_NODE_HEIGHT, 48 + titleLines * 18 + subtitleLines * 16 + 18);
      const height = Math.max(node.height || 0, baseHeight);
      return { width, height };
    }
  }
}

export function renderMockNodeShell(shape: FlowMockShape, borderColor: string, color: string, width: number, height: number) {
  const fill = 'var(--gantry-bg-primary)';

  switch (shape) {
    case 'pill':
      return (
        <div
          className="absolute inset-x-0 inset-y-1 rounded-full border"
          style={{ borderColor, background: fill }}
        />
      );
    case 'diamond':
      return (
        <svg className="absolute inset-0 h-full w-full overflow-visible" viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
          <polygon
            points={`${width / 2},6 ${width - 14},${height / 2} ${width / 2},${height - 6} 14,${height / 2}`}
            fill={fill}
            stroke={borderColor}
            strokeWidth="1.5"
          />
        </svg>
      );
    case 'note':
      return (
        <svg className="absolute inset-0 h-full w-full overflow-visible" viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
          <path
            d={`M 14 8 H ${width - 34} L ${width - 14} 28 V ${height - 14} Q ${width - 14} ${height - 8} ${width - 22} ${height - 8} H 22 Q 14 ${height - 8} 14 ${height - 16} Z`}
            fill={fill}
            stroke={borderColor}
            strokeWidth="1.5"
          />
          <path
            d={`M ${width - 34} 8 V 22 Q ${width - 34} 28 ${width - 28} 28 H ${width - 14} L ${width - 34} 8 Z`}
            fill={mockFoldFill(color)}
            stroke={borderColor}
            strokeWidth="1.5"
            strokeLinejoin="round"
          />
        </svg>
      );
    case 'box':
    default:
      return (
        <div
          className="absolute inset-0 rounded-2xl border"
          style={{ borderColor, background: fill }}
        />
      );
  }
}

export function mockContentClasses(shape: FlowMockShape): string {
  switch (shape) {
    case 'diamond':
      return 'items-center justify-center text-center px-10 py-6';
    case 'pill':
      return 'justify-center px-7 py-4';
    case 'note':
      return 'justify-between px-5 py-4 pr-12';
    case 'box':
    default:
      return 'justify-between px-4 py-3';
  }
}

export function mockContentStyle(shape: FlowMockShape, width: number): CSSProperties {
  switch (shape) {
    case 'diamond':
      return { maxWidth: Math.max(120, width * 0.44) };
    case 'pill':
      return { maxWidth: Math.max(140, width - 32) };
    case 'note':
      return { maxWidth: Math.max(140, width - 44) };
    case 'box':
    default:
      return { maxWidth: Math.max(140, width - 28) };
  }
}

export function getAbsolutePosition(
  node: FlowNode,
  nodeMap: Map<string, FlowNode>,
  visited: Set<string> = new Set()
): { x: number; y: number } {
  if (!node.parentId || visited.has(node.id)) return node.position;
  const parent = nodeMap.get(node.parentId);
  if (!parent) return node.position;
  visited.add(node.id);
  const base = getAbsolutePosition(parent, nodeMap, visited);
  return { x: base.x + node.position.x, y: base.y + node.position.y };
}

export function collectDescendants(nodeId: string, nodes: FlowNode[]): Set<string> {
  const descendants = new Set<string>();
  const queue = [nodeId];
  while (queue.length > 0) {
    const current = queue.shift()!;
    for (const n of nodes) {
      if (n.parentId === current && !descendants.has(n.id)) {
        descendants.add(n.id);
        queue.push(n.id);
      }
    }
  }
  return descendants;
}

export function edgePath(
  sourceAbsPos: { x: number; y: number },
  sourceSize: { width: number; height: number },
  targetAbsPos: { x: number; y: number },
  targetSize: { width: number; height: number }
): string {
  const x1 = sourceAbsPos.x + sourceSize.width;
  const y1 = sourceAbsPos.y + sourceSize.height / 2;
  const x2 = targetAbsPos.x;
  const y2 = targetAbsPos.y + targetSize.height / 2;
  const dx = Math.max(80, Math.abs(x2 - x1) * 0.45);
  return `M ${x1} ${y1} C ${x1 + dx} ${y1} ${x2 - dx} ${y2} ${x2} ${y2}`;
}

export function edgeLabelPosition(
  sourceAbsPos: { x: number; y: number },
  sourceSize: { width: number; height: number },
  targetAbsPos: { x: number; y: number },
  targetSize: { width: number; height: number }
) {
  return {
    x: (sourceAbsPos.x + sourceSize.width + targetAbsPos.x) / 2,
    y: (sourceAbsPos.y + targetAbsPos.y) / 2 + (sourceSize.height + targetSize.height) / 4 - 10,
  };
}

export function edgeOffsetTransform(
  sourceAbsPos: { x: number; y: number },
  sourceSize: { width: number; height: number },
  targetAbsPos: { x: number; y: number },
  targetSize: { width: number; height: number },
  offset: number
): string {
  const x1 = sourceAbsPos.x + sourceSize.width;
  const y1 = sourceAbsPos.y + sourceSize.height / 2;
  const x2 = targetAbsPos.x;
  const y2 = targetAbsPos.y + targetSize.height / 2;
  const dx = x2 - x1;
  const dy = y2 - y1;
  const length = Math.hypot(dx, dy) || 1;
  const nx = -dy / length;
  const ny = dx / length;
  return `translate(${nx * offset}, ${ny * offset})`;
}

export function autoArrangeNodes(spec: FlowSpec): FlowSpec {
  const topLevel = spec.nodes.filter((n) => !n.parentId);
  if (topLevel.length === 0) return spec;

  const HORIZONTAL_GAP = 80;
  const VERTICAL_GAP = 110;
  const PADDING = 48;

  // Build deduped edge graph (only among top-level nodes; ignore self-loops)
  const topLevelIds = new Set(topLevel.map((n) => n.id));
  const incoming = new Map<string, Set<string>>();
  const outgoing = new Map<string, Set<string>>();
  for (const node of topLevel) {
    incoming.set(node.id, new Set());
    outgoing.set(node.id, new Set());
  }
  for (const edge of spec.edges) {
    if (edge.source === edge.target) continue;
    if (!topLevelIds.has(edge.source) || !topLevelIds.has(edge.target)) continue;
    outgoing.get(edge.source)!.add(edge.target);
    incoming.get(edge.target)!.add(edge.source);
  }

  // Longest-path layering via Kahn's topological sort
  // level[n] = 1 + max(level of predecessors), defaults to 0 for roots
  const level = new Map<string, number>();
  const remainingInDegree = new Map<string, number>();
  for (const node of topLevel) {
    remainingInDegree.set(node.id, incoming.get(node.id)!.size);
  }

  const queue: string[] = [];
  for (const [id, count] of remainingInDegree) {
    if (count === 0) {
      level.set(id, 0);
      queue.push(id);
    }
  }

  while (queue.length > 0) {
    const id = queue.shift()!;
    const myLevel = level.get(id)!;
    for (const target of outgoing.get(id)!) {
      level.set(target, Math.max(level.get(target) ?? 0, myLevel + 1));
      remainingInDegree.set(target, remainingInDegree.get(target)! - 1);
      if (remainingInDegree.get(target) === 0) queue.push(target);
    }
  }

  // Any nodes still unleveled are trapped in cycles — assign based on known predecessors
  for (const node of topLevel) {
    if (level.has(node.id)) continue;
    const predLevels = [...incoming.get(node.id)!]
      .map((p) => level.get(p))
      .filter((l): l is number => l !== undefined);
    level.set(node.id, predLevels.length > 0 ? Math.max(...predLevels) + 1 : 0);
  }

  // Group nodes by level
  const byLevel = new Map<number, FlowNode[]>();
  for (const node of topLevel) {
    const lv = level.get(node.id)!;
    if (!byLevel.has(lv)) byLevel.set(lv, []);
    byLevel.get(lv)!.push(node);
  }
  const levels = [...byLevel.keys()].sort((a, b) => a - b);

  // Barycentric ordering within each level to reduce edge crossings
  // For each level past the first, sort nodes by the average index of their predecessors in the previous level
  const orderedByLevel = new Map<number, FlowNode[]>();
  for (let i = 0; i < levels.length; i++) {
    const lv = levels[i];
    const nodesAtLevel = byLevel.get(lv)!;

    if (i === 0) {
      orderedByLevel.set(lv, nodesAtLevel);
      continue;
    }

    const prevNodes = orderedByLevel.get(levels[i - 1])!;
    const prevIndex = new Map(prevNodes.map((n, idx) => [n.id, idx]));

    const sorted = [...nodesAtLevel].sort((a, b) => {
      const barycenter = (n: FlowNode) => {
        const preds = [...incoming.get(n.id)!]
          .map((id) => prevIndex.get(id))
          .filter((x): x is number => x !== undefined);
        if (preds.length === 0) return Number.POSITIVE_INFINITY;
        return preds.reduce((s, x) => s + x, 0) / preds.length;
      };
      return barycenter(a) - barycenter(b);
    });
    orderedByLevel.set(lv, sorted);
  }

  // Compute each level's max height and total width
  const levelHeights = new Map<number, number>();
  const levelWidths = new Map<number, number>();
  let maxLevelWidth = 0;
  for (const lv of levels) {
    const nodes = orderedByLevel.get(lv)!;
    const maxH = nodes.reduce((m, n) => Math.max(m, getNodeDimensions(n).height), 0);
    const totalW =
      nodes.reduce((s, n) => s + getNodeDimensions(n).width, 0) +
      Math.max(0, nodes.length - 1) * HORIZONTAL_GAP;
    levelHeights.set(lv, maxH);
    levelWidths.set(lv, totalW);
    if (totalW > maxLevelWidth) maxLevelWidth = totalW;
  }

  // Compute Y origin of each level (stacked top-to-bottom)
  const levelY = new Map<number, number>();
  let cumY = PADDING;
  for (const lv of levels) {
    levelY.set(lv, cumY);
    cumY += levelHeights.get(lv)! + VERTICAL_GAP;
  }

  // Assign final (x, y) to each node
  // X: centered within maxLevelWidth, sequential with HORIZONTAL_GAP
  // Y: centered vertically within level band so mixed heights align on midline
  const newPositions = new Map<string, { x: number; y: number }>();
  for (const lv of levels) {
    const nodes = orderedByLevel.get(lv)!;
    const rowY = levelY.get(lv)!;
    const levelH = levelHeights.get(lv)!;
    const startX = PADDING + (maxLevelWidth - levelWidths.get(lv)!) / 2;

    let x = startX;
    for (const node of nodes) {
      const dim = getNodeDimensions(node);
      newPositions.set(node.id, { x, y: rowY + (levelH - dim.height) / 2 });
      x += dim.width + HORIZONTAL_GAP;
    }
  }

  // Apply new positions only to top-level nodes; children keep their relative positions
  const updatedNodes = spec.nodes.map((node) => {
    const pos = newPositions.get(node.id);
    return pos ? { ...node, position: pos } : node;
  });

  return { ...spec, nodes: updatedNodes };
}
