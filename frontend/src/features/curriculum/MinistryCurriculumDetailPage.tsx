import { useMemo } from 'react'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { GitBranch, History, Layers, ListTree } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, CardHeader, CardTitle, Spinner, StatusPill } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getSubjectVersions, listSubjects, listTopicsBySubject } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { SubjectGraphView } from './SubjectGraphView'
import type { TopicListItem } from '@/types/api'

function formatDate(iso: string | undefined): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

function groupByUnit(topics: TopicListItem[]): Map<number, TopicListItem[]> {
  const byId = new Map(topics.map((t) => [t.id, t]))
  const groups = new Map<number, TopicListItem[]>()
  for (const t of topics) {
    // Only top-level topics anchor a unit group; subtopics render nested
    // under their parent within the same unit, matched via parentTopicId.
    if (t.parentTopicId && byId.has(t.parentTopicId)) continue
    if (!groups.has(t.unitNumber)) groups.set(t.unitNumber, [])
    groups.get(t.unitNumber)!.push(t)
  }
  for (const list of groups.values()) list.sort((a, b) => a.sequenceOrder - b.sequenceOrder)
  return groups
}

// Ministry curriculum browser detail view: everything about one promoted
// subject in a single place -- metadata, version lineage, the full
// unit/topic/subtopic tree, and the Neo4j knowledge graph -- so a
// ministry_admin doesn't have to hop between separate pages to review one
// subject end-to-end.
export function MinistryCurriculumDetailPage() {
  const { code } = useParams({ from: '/ministry/curriculum/$code' })
  const navigate = useNavigate()

  const subjectsQuery = useQuery({ queryKey: queryKeys.subjects(), queryFn: listSubjects })
  const subject = subjectsQuery.data?.find((s) => s.code === code)

  const topicsQuery = useQuery({
    queryKey: queryKeys.curriculumTopics(code),
    queryFn: () => listTopicsBySubject(code),
  })

  const versionsQuery = useQuery({
    queryKey: queryKeys.subjectVersions(code),
    queryFn: () => getSubjectVersions(code),
  })

  const unitGroups = useMemo(() => {
    if (!topicsQuery.data) return new Map<number, TopicListItem[]>()
    return groupByUnit(topicsQuery.data)
  }, [topicsQuery.data])

  const byId = useMemo(() => new Map((topicsQuery.data ?? []).map((t) => [t.id, t])), [topicsQuery.data])
  const subtopicsOf = (topicId: string) =>
    (topicsQuery.data ?? [])
      .filter((t) => t.parentTopicId === topicId)
      .sort((a, b) => a.sequenceOrder - b.sequenceOrder)

  return (
    <AppShell
      title={code}
      description={subject ? `${subject.nameEn} · Grade ${subject.gradeLevel} · ${subject.academicYear}` : 'Curriculum subject detail'}
    >
      <div className="space-y-6">
        <button
          type="button"
          onClick={() => void navigate({ to: '/ministry/curriculum' })}
          className="text-sm text-gray-500 hover:text-gray-700"
        >
          ← Back to curriculum by subject
        </button>

        {subjectsQuery.isLoading && (
          <div className="flex items-center gap-2 py-6 text-sm text-gray-500">
            <Spinner /> Loading subject...
          </div>
        )}
        {subjectsQuery.isError && (
          <Banner tone="error">{apiErrorMessage(subjectsQuery.error, 'Could not load subject metadata.')}</Banner>
        )}
        {!subjectsQuery.isLoading && !subject && !subjectsQuery.isError && (
          <Banner tone="error">No subject found for code "{code}".</Banner>
        )}

        {subject && (
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                {subject.code}
                <StatusPill tone={subject.isCurrent ? 'health' : 'neutral'}>
                  v{subject.version}
                  {subject.isCurrent ? ' · current' : ' · superseded'}
                </StatusPill>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
                <div>
                  <div className="text-gray-500">Grade / Year</div>
                  <div className="font-medium text-gray-900">
                    {subject.gradeLevel} · {subject.academicYear}
                  </div>
                </div>
                <div>
                  <div className="text-gray-500">Units</div>
                  <div className="font-medium text-gray-900">{subject.unitCount}</div>
                </div>
                <div>
                  <div className="text-gray-500">Topics / Subtopics</div>
                  <div className="font-medium text-gray-900">
                    {subject.topicCount} / {subject.subtopicCount}
                  </div>
                </div>
                <div>
                  <div className="text-gray-500">CLOs</div>
                  <div className="font-medium text-gray-900">{subject.cloCount}</div>
                </div>
                <div>
                  <div className="text-gray-500">Source File</div>
                  <div className="truncate font-medium text-gray-900" title={subject.fileName}>
                    {subject.fileName ?? '—'}
                  </div>
                </div>
                <div>
                  <div className="text-gray-500">Uploaded By</div>
                  <div className="font-medium text-gray-900">{subject.uploadedByName ?? '—'}</div>
                </div>
                <div>
                  <div className="text-gray-500">Approved</div>
                  <div className="font-medium text-gray-900">{formatDate(subject.approvedAt)}</div>
                </div>
                <div>
                  <div className="text-gray-500">MoE Code</div>
                  <div className="font-medium text-gray-900">{subject.moeCode ?? '—'}</div>
                </div>
              </div>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <History className="h-4 w-4" /> Version Lineage
            </CardTitle>
          </CardHeader>
          <CardContent>
            {versionsQuery.isLoading && (
              <div className="flex items-center gap-2 py-2 text-sm text-gray-500">
                <Spinner /> Loading versions...
              </div>
            )}
            {versionsQuery.isError && (
              <Banner tone="error">{apiErrorMessage(versionsQuery.error, 'Could not load version history.')}</Banner>
            )}
            {versionsQuery.data && (
              <ol className="space-y-2 text-sm">
                {versionsQuery.data.map((v) => (
                  <li key={v.code} className="flex items-center gap-3 rounded-lg border border-gray-100 px-3 py-2">
                    <StatusPill tone={v.isCurrent ? 'health' : 'neutral'}>v{v.version}</StatusPill>
                    <span className="font-medium text-gray-900">{v.code}</span>
                    <span className="text-gray-500">{v.academicYear}</span>
                    {v.supersededAt && (
                      <span className="ml-auto text-xs text-gray-400">Superseded {formatDate(v.supersededAt)}</span>
                    )}
                  </li>
                ))}
              </ol>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ListTree className="h-4 w-4" /> Units & Topics
            </CardTitle>
          </CardHeader>
          <CardContent>
            {topicsQuery.isLoading && (
              <div className="flex items-center gap-2 py-2 text-sm text-gray-500">
                <Spinner /> Loading topics...
              </div>
            )}
            {topicsQuery.isError && (
              <Banner tone="error">{apiErrorMessage(topicsQuery.error, 'Could not load topics.')}</Banner>
            )}
            {topicsQuery.data && (
              <div className="max-h-[420px] space-y-4 overflow-y-auto pr-2">
                {[...unitGroups.entries()]
                  .sort(([a], [b]) => a - b)
                  .map(([unitNumber, topics]) => (
                    <div key={unitNumber}>
                      <div className="mb-1 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-gray-400">
                        <Layers className="h-3.5 w-3.5" /> Unit {unitNumber}
                      </div>
                      <ul className="space-y-1">
                        {topics.map((t) => (
                          <li key={t.id}>
                            <div className="rounded-md px-2 py-1 text-sm text-gray-800 hover:bg-gray-50">{t.titleEn}</div>
                            {subtopicsOf(t.id).length > 0 && (
                              <ul className="ml-4 border-l border-gray-100 pl-3">
                                {subtopicsOf(t.id).map((s) => (
                                  <li key={s.id} className="rounded-md px-2 py-0.5 text-xs text-gray-500 hover:bg-gray-50">
                                    {s.titleEn}
                                  </li>
                                ))}
                              </ul>
                            )}
                          </li>
                        ))}
                      </ul>
                    </div>
                  ))}
                {byId.size === 0 && <p className="text-sm text-gray-500">No topics found for this subject.</p>}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <GitBranch className="h-4 w-4" /> Knowledge Graph
            </CardTitle>
          </CardHeader>
          <CardContent>
            <SubjectGraphView code={code} height="h-[600px]" />
          </CardContent>
        </Card>
      </div>
    </AppShell>
  )
}
