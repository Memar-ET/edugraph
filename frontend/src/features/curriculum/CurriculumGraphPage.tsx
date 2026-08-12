import { useNavigate, useParams } from '@tanstack/react-router'

import { AppShell } from '@components/layout'
import { SubjectGraphView } from './SubjectGraphView'

// Feature: "view the Neo4j graph of an uploaded curriculum" -- fetches
// the subject's Subject->Unit->Topic->Subtopic[->CLO] subtree plus
// cross-grade HAS_PREREQUISITE edges and renders it with ReactFlow + a
// dagre auto-layout. Rendering itself lives in SubjectGraphView, shared
// with the Ministry subject detail page so the graph isn't duplicated.
export function CurriculumGraphPage() {
  const { code } = useParams({ from: '/curriculum/subjects/$code/graph' })
  const navigate = useNavigate()

  return (
    <AppShell
      title={`Knowledge Graph — ${code}`}
      description="Subject → Unit → Topic → Subtopic structure, mirrored from Neo4j, with cross-grade prerequisite edges."
    >
      <div className="space-y-4">
        <button
          type="button"
          onClick={() => void navigate({ to: '/curriculum' })}
          className="text-sm text-gray-500 hover:text-gray-700"
        >
          ← Back to curriculum dashboard
        </button>

        <SubjectGraphView code={code} />
      </div>
    </AppShell>
  )
}
