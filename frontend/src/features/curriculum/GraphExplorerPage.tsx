import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Search, Network, ArrowRight } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Spinner, Banner, EmptyState } from '@components/ui'
import { listSubjects } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function CurriculumGraphExplorerPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')

  const { data: subjects, isLoading, isError } = useQuery({
    queryKey: queryKeys.subjects(),
    queryFn: listSubjects,
    select: (items) => items.filter((s) => s.isCurrent !== false),
  })

  const filtered = (subjects ?? []).filter((s) =>
    !search ||
    s.code.toLowerCase().includes(search.toLowerCase()) ||
    (s.nameEn ?? '').toLowerCase().includes(search.toLowerCase()),
  )

  const openGraph = (code: string) => {
    void navigate({ to: '/curriculum/subjects/$code/graph', params: { code } })
  }

  return (
    <AppShell
      title="Curriculum Graph Explorer"
      description="Visualize the Neo4j prerequisite dependency graph (DAG) for any subject and grade."
    >
      <div className="space-y-5">
        {/* Info Banner */}
        <div className="rounded-2xl border border-teal-200 bg-teal-50 p-4 text-teal-900 text-xs">
          <p className="font-bold">Real-time Neo4j Graph Visualization</p>
          <p className="mt-1 text-teal-700">
            Select a subject below to open its interactive prerequisite dependency graph. Nodes represent topics and subtopics; edges show prerequisite relationships. Use the graph viewer to pan, zoom, and toggle CLO visibility.
          </p>
        </div>

        {/* Search */}
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400" />
          <input
            type="text"
            placeholder="Search subjects..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-xl border border-slate-200 bg-white py-2 pl-8 pr-3 text-xs text-slate-800 shadow-sm focus:border-teal-500 focus:outline-none"
          />
        </div>

        {isLoading && (
          <div className="flex items-center gap-2 py-8 text-xs text-slate-500">
            <Spinner /> Loading subjects...
          </div>
        )}
        {isError && <Banner tone="error">Could not load curriculum subjects. Ensure the database is reachable.</Banner>}
        {!isLoading && filtered.length === 0 && (
          <EmptyState
            icon={Network}
            title="No subjects found"
            description="Upload and approve curriculum documents to populate the graph."
          />
        )}

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((s) => (
            <button
              key={s.code}
              type="button"
              onClick={() => openGraph(s.code)}
              className="group flex items-center justify-between rounded-2xl border border-slate-200 bg-white p-5 shadow-sm text-left transition-all hover:border-teal-500 hover:shadow-md"
            >
              <div className="min-w-0">
                <span className="font-mono text-xs font-bold text-teal-700 block">{s.code}</span>
                <span className="text-sm font-bold text-slate-900 block mt-0.5 truncate">
                  {s.nameEn ?? s.code}
                </span>
                <span className="text-xs text-slate-500 mt-1 block">Grade {s.gradeLevel}</span>
              </div>
              <div className="ml-3 shrink-0 rounded-full bg-teal-50 p-2 group-hover:bg-teal-100 transition-colors">
                <ArrowRight className="h-4 w-4 text-teal-700" />
              </div>
            </button>
          ))}
        </div>
      </div>
    </AppShell>
  )
}
