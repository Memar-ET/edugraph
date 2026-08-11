import { useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowRight, CheckCircle2, RefreshCw } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Input, Label, Select, Spinner, StatusPill } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import {
  addTopicPrerequisite,
  listTopicPrerequisites,
  listTopicsBySubject,
  resyncPrerequisites,
  validatePrerequisite,
} from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { useAuthStore } from '@stores/auth.store'
import type { PrerequisiteInferMethod, TopicListItem } from '@/types/api'

const INFER_METHODS: { value: PrerequisiteInferMethod; label: string }[] = [
  { value: 'manual', label: 'Manual (auto-validated)' },
  { value: 'explicit', label: 'Explicit in source (auto-validated)' },
  { value: 'moe_document', label: 'MoE document (auto-validated)' },
  { value: 'ai_inferred', label: 'AI-inferred (needs validation)' },
]

// Topics + subtopics come back as a flat, subject-scoped list (no general
// "browse the curriculum" endpoint exists) -- this indents a subtopic
// under its parent using parentTopicId, purely for readability in the picker.
function topicLabel(topic: TopicListItem, byId: Map<string, TopicListItem>): string {
  const indent = topic.parentTopicId && byId.has(topic.parentTopicId) ? '— ' : ''
  return `U${topic.unitNumber} · ${indent}${topic.titleEn}`
}

export function PrerequisitesPage() {
  const role = useAuthStore((s) => s.user?.role)
  const queryClient = useQueryClient()

  const [subjectCode, setSubjectCode] = useState('')
  const [activeSubject, setActiveSubject] = useState<string | null>(null)
  const [topicId, setTopicId] = useState('')
  const [prereqTopicId, setPrereqTopicId] = useState('')
  const [weight, setWeight] = useState('1')
  const [inferMethod, setInferMethod] = useState<PrerequisiteInferMethod>('manual')

  const [addError, setAddError] = useState<string | null>(null)
  const [adding, setAdding] = useState(false)
  const [validatingId, setValidatingId] = useState<string | null>(null)
  const [resyncing, setResyncing] = useState(false)
  const [resyncResult, setResyncResult] = useState<{ synced: number; failed: number } | null>(null)
  const [resyncError, setResyncError] = useState<string | null>(null)

  const topicsQuery = useQuery({
    queryKey: queryKeys.curriculumTopics(activeSubject ?? ''),
    queryFn: () => listTopicsBySubject(activeSubject!),
    enabled: Boolean(activeSubject),
  })

  const topicsById = useMemo(() => {
    const map = new Map<string, TopicListItem>()
    for (const t of topicsQuery.data ?? []) map.set(t.id, t)
    return map
  }, [topicsQuery.data])

  const prereqsQuery = useQuery({
    queryKey: queryKeys.topicPrerequisites(topicId),
    queryFn: () => listTopicPrerequisites(topicId),
    enabled: Boolean(topicId),
  })

  const loadSubject = () => {
    const code = subjectCode.trim().toUpperCase()
    if (!code) return
    setActiveSubject(code)
    setTopicId('')
    setPrereqTopicId('')
  }

  const handleAdd = async () => {
    if (!topicId || !prereqTopicId) return
    setAdding(true)
    setAddError(null)
    try {
      await addTopicPrerequisite(topicId, {
        prerequisiteTopicId: prereqTopicId,
        weight: Number(weight) || 1,
        inferMethod,
      })
      await queryClient.invalidateQueries({ queryKey: queryKeys.topicPrerequisites(topicId) })
      setPrereqTopicId('')
    } catch (err) {
      setAddError(apiErrorMessage(err, 'Could not add the prerequisite link.'))
    } finally {
      setAdding(false)
    }
  }

  const handleValidate = async (prereqTopicIdToValidate: string) => {
    setValidatingId(prereqTopicIdToValidate)
    try {
      await validatePrerequisite(topicId, prereqTopicIdToValidate)
      await queryClient.invalidateQueries({ queryKey: queryKeys.topicPrerequisites(topicId) })
    } catch (err) {
      setAddError(apiErrorMessage(err, 'Could not validate the link.'))
    } finally {
      setValidatingId(null)
    }
  }

  const handleResync = async () => {
    setResyncing(true)
    setResyncError(null)
    setResyncResult(null)
    try {
      const res = await resyncPrerequisites()
      setResyncResult(res)
    } catch (err) {
      setResyncError(apiErrorMessage(err, 'Resync failed.'))
    } finally {
      setResyncing(false)
    }
  }

  const topics = topicsQuery.data ?? []
  const selectedTopic = topicsById.get(topicId)

  return (
    <AppShell
      title="Topic Prerequisites"
      description="Manage the prerequisite graph that feeds gap-analysis root causes and study-plan ordering."
    >
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Load a subject's topics</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap items-end gap-2">
              <div>
                <Label htmlFor="subjectCode">Subject code</Label>
                <Input
                  id="subjectCode"
                  value={subjectCode}
                  onChange={(e) => setSubjectCode(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && loadSubject()}
                  placeholder="BIO-G9"
                  className="w-40"
                />
              </div>
              <Button variant="secondary" onClick={loadSubject}>
                Load topics
              </Button>
            </div>
            {topicsQuery.isLoading && (
              <div className="mt-3 flex items-center gap-2 text-sm text-gray-500">
                <Spinner /> Loading topics…
              </div>
            )}
            {topicsQuery.isError && (
              <Banner tone="error" className="mt-3">
                {apiErrorMessage(topicsQuery.error, 'Could not load topics for that subject.')}
              </Banner>
            )}
            {activeSubject && !topicsQuery.isLoading && !topicsQuery.isError && topics.length === 0 && (
              <p className="mt-3 text-sm text-gray-500">No topics found for {activeSubject}.</p>
            )}
          </CardContent>
        </Card>

        {topics.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Pick a topic</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <Select value={topicId} onChange={(e) => setTopicId(e.target.value)}>
                <option value="">Select a topic…</option>
                {topics.map((t) => (
                  <option key={t.id} value={t.id}>
                    {topicLabel(t, topicsById)}
                  </option>
                ))}
              </Select>

              {topicId && (
                <div className="space-y-4 border-t border-gray-100 pt-4">
                  <h3 className="text-sm font-medium text-gray-900">
                    Prerequisites for {selectedTopic ? topicLabel(selectedTopic, topicsById) : '…'}
                  </h3>

                  {prereqsQuery.isLoading && (
                    <div className="flex items-center gap-2 text-sm text-gray-500">
                      <Spinner /> Loading prerequisites…
                    </div>
                  )}
                  {prereqsQuery.isError && (
                    <Banner tone="error">
                      {apiErrorMessage(prereqsQuery.error, 'Could not load prerequisites.')}
                    </Banner>
                  )}

                  {prereqsQuery.data && prereqsQuery.data.length === 0 && (
                    <p className="text-sm text-gray-500">No prerequisites recorded for this topic yet.</p>
                  )}

                  {prereqsQuery.data && prereqsQuery.data.length > 0 && (
                    <div className="overflow-x-auto">
                      <table className="w-full min-w-[560px] text-left text-sm">
                        <thead>
                          <tr className="border-b border-gray-200 text-xs font-medium uppercase tracking-wide text-gray-500">
                            <th className="py-2 pr-4">Requires</th>
                            <th className="py-2 pr-4">Grade</th>
                            <th className="py-2 pr-4">Weight</th>
                            <th className="py-2 pr-4">Source</th>
                            <th className="py-2 pr-4">Status</th>
                            <th className="py-2" />
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-100">
                          {prereqsQuery.data.map((link) => (
                            <tr key={link.prerequisiteTopicId}>
                              <td className="py-3 pr-4 font-medium text-gray-900">
                                {link.prerequisiteTitle}
                                {link.isCrossGrade && (
                                  <StatusPill tone="seal" className="ml-2">
                                    Cross-grade
                                  </StatusPill>
                                )}
                              </td>
                              <td className="py-3 pr-4 text-gray-500">{link.prerequisiteGrade}</td>
                              <td className="py-3 pr-4 text-gray-500">{link.weight.toFixed(2)}</td>
                              <td className="py-3 pr-4 text-gray-500">{link.inferMethod}</td>
                              <td className="py-3 pr-4">
                                <StatusPill tone={link.isValidated ? 'health' : 'alert'}>
                                  {link.isValidated ? 'Validated' : 'Inferred'}
                                </StatusPill>
                              </td>
                              <td className="py-3 text-right">
                                {!link.isValidated && (
                                  <Button
                                    variant="secondary"
                                    size="sm"
                                    isLoading={validatingId === link.prerequisiteTopicId}
                                    onClick={() => void handleValidate(link.prerequisiteTopicId)}
                                  >
                                    <CheckCircle2 className="h-4 w-4" aria-hidden />
                                    Validate
                                  </Button>
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}

                  <div className="rounded-lg border border-gray-100 bg-gray-50 p-4">
                    <h4 className="mb-3 text-sm font-medium text-gray-900">Add a prerequisite</h4>
                    {addError && <Banner tone="error" className="mb-3">{addError}</Banner>}
                    <div className="flex flex-wrap items-end gap-2">
                      <div className="min-w-[220px] flex-1">
                        <Label htmlFor="prereqTopic">Requires (prerequisite topic)</Label>
                        <Select
                          id="prereqTopic"
                          value={prereqTopicId}
                          onChange={(e) => setPrereqTopicId(e.target.value)}
                        >
                          <option value="">Select a topic…</option>
                          {topics
                            .filter((t) => t.id !== topicId)
                            .map((t) => (
                              <option key={t.id} value={t.id}>
                                {topicLabel(t, topicsById)}
                              </option>
                            ))}
                        </Select>
                      </div>
                      <div>
                        <Label htmlFor="weight">Weight</Label>
                        <Input
                          id="weight"
                          type="number"
                          min={0}
                          max={1}
                          step={0.1}
                          value={weight}
                          onChange={(e) => setWeight(e.target.value)}
                          className="w-24"
                        />
                      </div>
                      <div>
                        <Label htmlFor="inferMethod">Source</Label>
                        <Select
                          id="inferMethod"
                          value={inferMethod}
                          onChange={(e) => setInferMethod(e.target.value as PrerequisiteInferMethod)}
                          className="w-56"
                        >
                          {INFER_METHODS.map((m) => (
                            <option key={m.value} value={m.value}>
                              {m.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                      <Button isLoading={adding} disabled={!prereqTopicId} onClick={() => void handleAdd()}>
                        <ArrowRight className="h-4 w-4" aria-hidden />
                        Add link
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {role === 'ministry_admin' && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Neo4j sync</CardTitle>
              <p className="mt-1 text-sm text-gray-500">
                Re-mirrors every prerequisite link Postgres doesn't know is current in Neo4j (failed best-effort
                syncs, or rows that predate the sync-tracking column).
              </p>
            </CardHeader>
            <CardContent className="space-y-3">
              {resyncError && <Banner tone="error">{resyncError}</Banner>}
              {resyncResult && (
                <Banner tone={resyncResult.failed > 0 ? 'warning' : 'success'}>
                  {resyncResult.synced} synced, {resyncResult.failed} failed.
                </Banner>
              )}
              <Button variant="secondary" isLoading={resyncing} onClick={() => void handleResync()}>
                <RefreshCw className="h-4 w-4" aria-hidden />
                Resync to Neo4j
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </AppShell>
  )
}
