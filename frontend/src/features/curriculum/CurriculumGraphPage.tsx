import { useMemo, useState } from 'react'
import type { CSSProperties } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import dagre from 'dagre'
import ReactFlow, {
  Background,
  Controls,
  MarkerType,
  MiniMap,
  Position,
  type Edge,
  type Node,
} from 'reactflow'
import 'reactflow/dist/style.css'

import { AppShell } from '@components/layout'
import { Banner, Spinner } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getSubjectGraph } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { GraphEdge, GraphNode, GraphNodeType } from '@/types/api'

const NODE_WIDTH = 190
const NODE_HEIGHT = 44

const NODE_STYLE: Record<GraphNodeType, CSSProperties> = {
  subject: { background: '#111827', color: '#fff', border: '1px solid #111827', fontWeight: 600 },
  unit: { background: '#eff6ff', color: '#1e3a8a', border: '1px solid #93c5fd', fontWeight: 500 },
  topic: { background: '#f0fdf4', color: '#14532d', border: '1px solid #86efac' },
  clo: { background: '#fffbeb', color: '#78350f', border: '1px solid #fde68a', fontSize: 11 },
}

const SUBTOPIC_STYLE: CSSProperties = { background: '#f0fdf4', color: '#166534', border: '1px solid #bbf7d0', fontSize: 12 }
const EXTERNAL_STYLE: CSSProperties = { background: '#f9fafb', color: '#6b7280', border: '1px dashed #d1d5db', fontStyle: 'italic' }

function nodeStyle(n: GraphNode): CSSProperties {
  if (n.external) return EXTERNAL_STYLE
  if (n.type === 'topic' && n.isSubtopic) return SUBTOPIC_STYLE
  return NODE_STYLE[n.type]
}

function nodeLabel(n: GraphNode): string {
  if (n.type === 'topic' && n.external) return `${n.label} (G${n.gradeLevel ?? '?'}${n.subjectCode ? `, ${n.subjectCode}` : ''})`
  return n.label
}

// Dagre computes a hierarchical layout (top-to-bottom); ReactFlow just
// renders whatever positions we hand it -- neither library does both.
function layout(rawNodes: GraphNode[], rawEdges: GraphEdge[]): { nodes: Node[]; edges: Edge[] } {
  const g = new dagre.graphlib.Graph()
  g.setDefaultEdgeLabel(() => ({}))
  g.setGraph({ rankdir: 'TB', nodesep: 32, ranksep: 70 })

  for (const n of rawNodes) g.setNode(n.id, { width: NODE_WIDTH, height: NODE_HEIGHT })
  for (const e of rawEdges) g.setEdge(e.source, e.target)

  dagre.layout(g)

  const nodes: Node[] = rawNodes.map((n) => {
    const pos = g.node(n.id)
    return {
      id: n.id,
      data: { label: nodeLabel(n) },
      position: { x: (pos?.x ?? 0) - NODE_WIDTH / 2, y: (pos?.y ?? 0) - NODE_HEIGHT / 2 },
      style: { ...nodeStyle(n), width: NODE_WIDTH, borderRadius: 8, padding: '6px 10px', fontSize: n.type === 'clo' ? 11 : 13 },
      sourcePosition: Position.Bottom,
      targetPosition: Position.Top,
    }
  })

  const edges: Edge[] = rawEdges.map((e, i) => {
    const isPrereq = e.type === 'HAS_PREREQUISITE'
    return {
      id: `${e.source}-${e.target}-${i}`,
      source: e.source,
      target: e.target,
      type: 'smoothstep',
      animated: isPrereq,
      style: isPrereq ? { stroke: '#f59e0b', strokeDasharray: '4 3' } : { stroke: '#d1d5db' },
      markerEnd: isPrereq ? { type: MarkerType.ArrowClosed, color: '#f59e0b' } : undefined,
      zIndex: isPrereq ? 10 : 0,
    }
  })

  return { nodes, edges }
}

// Feature: "view the Neo4j graph of an uploaded curriculum" -- fetches
// the subject's Subject->Unit->Topic->Subtopic[->CLO] subtree plus
// cross-grade HAS_PREREQUISITE edges from GET .../subjects/{code}/graph
// and renders it with ReactFlow + a dagre auto-layout (neither library
// does both on its own).
export function CurriculumGraphPage() {
  const { code } = useParams({ from: '/curriculum/subjects/$code/graph' })
  const navigate = useNavigate()
  const [includeClos, setIncludeClos] = useState(false)

  const graphQuery = useQuery({
    queryKey: queryKeys.subjectGraph(code, includeClos),
    queryFn: () => getSubjectGraph(code, includeClos),
  })

  const [includeSubtopics, setIncludeSubtopics] = useState(false)

  // Subtopics alone roughly quadruple BIO-grade node counts (445 vs 109
  // top-level topics across the whole dataset) -- off by default, same as
  // CLOs, so the initial view is legible instead of dagre laying out 100+
  // nodes across one very wide rank and fitView zooming out to near-nothing.
  const filtered = useMemo(() => {
    if (!graphQuery.data) return null
    if (includeSubtopics) return graphQuery.data
    const keepIds = new Set(
      graphQuery.data.nodes.filter((n) => !(n.type === 'topic' && n.isSubtopic)).map((n) => n.id),
    )
    return {
      nodes: graphQuery.data.nodes.filter((n) => keepIds.has(n.id)),
      edges: graphQuery.data.edges.filter(
        (e) => e.type !== 'HAS_SUBTOPIC' && keepIds.has(e.source) && keepIds.has(e.target),
      ),
    }
  }, [graphQuery.data, includeSubtopics])

  const { nodes, edges } = useMemo(() => {
    if (!filtered) return { nodes: [], edges: [] }
    return layout(filtered.nodes, filtered.edges)
  }, [filtered])

  return (
    <AppShell
      title={`Knowledge Graph — ${code}`}
      description="Subject → Unit → Topic → Subtopic structure, mirrored from Neo4j, with cross-grade prerequisite edges."
      actions={
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 text-sm text-gray-600">
            <input
              type="checkbox"
              checked={includeSubtopics}
              onChange={(e) => setIncludeSubtopics(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300"
            />
            Show subtopics
          </label>
          <label className="flex items-center gap-2 text-sm text-gray-600">
            <input
              type="checkbox"
              checked={includeClos}
              onChange={(e) => setIncludeClos(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300"
            />
            Show CLOs
          </label>
        </div>
      }
    >
      <div className="space-y-4">
        <button
          type="button"
          onClick={() => void navigate({ to: '/curriculum' })}
          className="text-sm text-gray-500 hover:text-gray-700"
        >
          ← Back to curriculum dashboard
        </button>

        {graphQuery.isLoading && (
          <div className="flex items-center gap-2 py-12 text-sm text-gray-500">
            <Spinner /> Loading graph…
          </div>
        )}
        {graphQuery.isError && (
          <Banner tone="error">{apiErrorMessage(graphQuery.error, `Could not load the graph for ${code}.`)}</Banner>
        )}

        {graphQuery.data && (
          <>
            <div className="flex flex-wrap items-center gap-4 rounded-lg border border-gray-100 bg-white px-4 py-2 text-xs text-gray-600">
              <Legend swatch="#111827" label="Subject" />
              <Legend swatch="#eff6ff" border="#93c5fd" label="Unit" />
              <Legend swatch="#f0fdf4" border="#86efac" label="Topic" />
              {includeSubtopics && <Legend swatch="#f0fdf4" border="#bbf7d0" label="Subtopic" />}
              {includeClos && <Legend swatch="#fffbeb" border="#fde68a" label="CLO" />}
              <Legend swatch="#f9fafb" border="#d1d5db" dashed label="Other grade/subject" />
              <span className="flex items-center gap-1">
                <span className="inline-block h-0 w-6 border-t-2 border-dashed" style={{ borderColor: '#f59e0b' }} />
                Prerequisite
              </span>
              <span className="ml-auto text-gray-400">
                {nodes.length} nodes · {edges.length} edges
                {!includeSubtopics && ` (of ${graphQuery.data.nodes.length} total -- subtopics hidden)`}
              </span>
            </div>

            <div className="h-[calc(100vh-260px)] min-h-[500px] rounded-lg border border-gray-200 bg-gray-50">
              <ReactFlow nodes={nodes} edges={edges} fitView minZoom={0.05} proOptions={{ hideAttribution: true }}>
                <Background />
                <Controls />
                <MiniMap pannable zoomable />
              </ReactFlow>
            </div>
          </>
        )}
      </div>
    </AppShell>
  )
}

function Legend({ swatch, border, label, dashed }: { swatch: string; border?: string; label: string; dashed?: boolean }) {
  return (
    <span className="flex items-center gap-1.5">
      <span
        className="h-3 w-3 rounded"
        style={{ background: swatch, border: `1px ${dashed ? 'dashed' : 'solid'} ${border ?? swatch}` }}
      />
      {label}
    </span>
  )
}
