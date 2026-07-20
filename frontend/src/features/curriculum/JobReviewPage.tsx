import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useEffect, useState } from 'react'

import { apiErrorMessage } from '@lib/api/client'
import { approveCurriculumJob, fetchCurriculumFileBlobUrl, getCurriculumJob } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, Input, Spinner } from '@components/ui'
import { AppShell } from '@components/layout'
import type { ApproveResponse, ParsedTopic, ParsedUnit } from '@/types/api'

const IN_PROGRESS_STATUSES = new Set(['pending', 'parsing'])

function linesToList(text: string): string[] {
  return text
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
}

export function JobReviewPage() {
  const { jobId } = useParams({ from: '/curriculum/jobs/$jobId' })
  const navigate = useNavigate()

  const { data: job, isLoading, error } = useQuery({
    queryKey: queryKeys.curriculumJob(jobId),
    queryFn: () => getCurriculumJob(jobId),
    refetchInterval: (query) => (IN_PROGRESS_STATUSES.has(query.state.data?.status ?? '') ? 3000 : false),
  })

  const [units, setUnits] = useState<ParsedUnit[] | null>(null)
  const [dirty, setDirty] = useState(false)
  const [approving, setApproving] = useState(false)
  const [approveError, setApproveError] = useState<string | null>(null)
  const [approveResult, setApproveResult] = useState<ApproveResponse | null>(null)
  const [fileUrl, setFileUrl] = useState<string | null>(null)

  // Seed the editable copy once the parsed structure first arrives -- don't
  // clobber in-progress officer edits on subsequent poll refetches.
  useEffect(() => {
    if (units === null && job?.parsedStructure) {
      setUnits(structuredClone(job.parsedStructure.units))
    }
  }, [job, units])

  useEffect(() => {
    return () => {
      if (fileUrl) URL.revokeObjectURL(fileUrl)
    }
  }, [fileUrl])

  const handleViewFile = async () => {
    try {
      const url = await fetchCurriculumFileBlobUrl(jobId)
      setFileUrl(url)
      window.open(url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      setApproveError(apiErrorMessage(err, 'Could not load the original file.'))
    }
  }

  const updateUnitTitle = (unitIdx: number, title: string) => {
    setDirty(true)
    setUnits((prev) => {
      if (!prev) return prev
      const next = [...prev]
      next[unitIdx] = { ...next[unitIdx]!, titleEn: title }
      return next
    })
  }

  const updateTopic = (unitIdx: number, topicIdx: number, patch: Partial<ParsedTopic>) => {
    setDirty(true)
    setUnits((prev) => {
      if (!prev) return prev
      const next = [...prev]
      const unit = { ...next[unitIdx]! }
      const topics = [...unit.topics]
      topics[topicIdx] = { ...topics[topicIdx]!, ...patch }
      unit.topics = topics
      next[unitIdx] = unit
      return next
    })
  }

  const removeTopic = (unitIdx: number, topicIdx: number) => {
    setDirty(true)
    setUnits((prev) => {
      if (!prev) return prev
      const next = [...prev]
      const unit = { ...next[unitIdx]! }
      unit.topics = unit.topics.filter((_, i) => i !== topicIdx)
      next[unitIdx] = unit
      return next
    })
  }

  const addTopic = (unitIdx: number) => {
    setDirty(true)
    setUnits((prev) => {
      if (!prev) return prev
      const next = [...prev]
      const unit = { ...next[unitIdx]! }
      unit.topics = [
        ...unit.topics,
        { sequenceOrder: unit.topics.length + 1, titleEn: 'New topic', keyConcepts: [], learningOutcomes: [], clos: [] },
      ]
      next[unitIdx] = unit
      return next
    })
  }

  const removeUnit = (unitIdx: number) => {
    setDirty(true)
    setUnits((prev) => (prev ? prev.filter((_, i) => i !== unitIdx) : prev))
  }

  const addUnit = () => {
    setDirty(true)
    setUnits((prev) => [
      ...(prev ?? []),
      { number: (prev?.length ?? 0) + 1, titleEn: 'New unit', topics: [] },
    ])
  }

  const handleApprove = async () => {
    if (!units) return
    setApproving(true)
    setApproveError(null)
    try {
      const normalized: ParsedUnit[] = units.map((u, i) => ({
        ...u,
        number: i + 1,
        topics: u.topics.map((t, j) => ({ ...t, sequenceOrder: j + 1 })),
      }))
      const result = await approveCurriculumJob(jobId, dirty ? { parsedStructure: { units: normalized } } : {})
      setApproveResult(result)
    } catch (err) {
      setApproveError(apiErrorMessage(err, 'Approval failed. Please try again.'))
    } finally {
      setApproving(false)
    }
  }

  return (
    <AppShell title="Review curriculum job" description="Step 3: review and approve the AI-extracted structure.">
      <div className="mx-auto max-w-3xl space-y-4">
        <Button variant="ghost" size="sm" onClick={() => void navigate({ to: '/curriculum/upload' })}>
          ← Upload another document
        </Button>

        {isLoading && (
          <div className="flex items-center gap-2 text-gray-500">
            <Spinner /> Loading job…
          </div>
        )}
        {error && <Banner tone="error">{apiErrorMessage(error, 'Could not load this job.')}</Banner>}

        {job && (
          <>
            <Card>
              <CardHeader>
                <CardTitle>
                  {job.subjectCode} · Grade {job.gradeLevel} · {job.academicYear}
                </CardTitle>
                <p className="mt-1 text-sm text-gray-500">{job.fileName}</p>
              </CardHeader>
              <CardContent className="space-y-3">
                <StatusBanner status={job.status} error={job.error} />

                <Button variant="secondary" size="sm" onClick={() => void handleViewFile()}>
                  View original file
                </Button>
              </CardContent>
            </Card>

            {IN_PROGRESS_STATUSES.has(job.status) && (
              <Banner tone="info">
                <div className="flex items-center gap-2">
                  <Spinner className="h-4 w-4" /> Parsing is in progress — this page refreshes automatically.
                </div>
              </Banner>
            )}

            {approveResult && (
              <Banner tone={approveResult.graphSynced ? 'success' : 'warning'}>
                Approved. {approveResult.unitsPromoted} unit(s), {approveResult.topicsPromoted} topic(s),{' '}
                {approveResult.closPromoted} CLO(s) promoted.{' '}
                {approveResult.graphSynced
                  ? 'Knowledge graph synced.'
                  : `Knowledge graph sync failed (${approveResult.graphSyncError ?? 'unknown error'}) — Postgres promotion still succeeded; re-approving will retry the graph sync.`}
              </Banner>
            )}

            {approveError && <Banner tone="error">{approveError}</Banner>}

            {units && !approveResult && (
              <>
                <div className="space-y-4">
                  {units.map((unit, unitIdx) => (
                    <Card key={unitIdx}>
                      <CardHeader className="flex flex-row items-center justify-between gap-2">
                        <Input
                          value={unit.titleEn}
                          onChange={(e) => updateUnitTitle(unitIdx, e.target.value)}
                          className="text-base font-semibold"
                        />
                        <Button variant="ghost" size="sm" onClick={() => removeUnit(unitIdx)}>
                          Remove unit
                        </Button>
                      </CardHeader>
                      {unit.metadata && Object.keys(unit.metadata).length > 0 && (
                        <div className="mx-6 mb-2 grid grid-cols-2 gap-x-4 gap-y-1 rounded-md bg-gray-50 p-3 text-xs text-gray-600 sm:grid-cols-3">
                          {Object.entries(unit.metadata).map(([key, value]) => (
                            <div key={key}>
                              <span className="font-medium text-gray-500">{key}:</span> {value}
                            </div>
                          ))}
                        </div>
                      )}
                      <CardContent className="space-y-4">
                        {unit.topics.map((topic, topicIdx) => (
                          <div key={topicIdx} className="rounded-md border border-gray-200 p-3 space-y-2">
                            <div className="flex items-center justify-between gap-2">
                              <Input
                                value={topic.titleEn}
                                onChange={(e) =>
                                  updateTopic(unitIdx, topicIdx, { titleEn: e.target.value })
                                }
                              />
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => removeTopic(unitIdx, topicIdx)}
                              >
                                Remove
                              </Button>
                            </div>

                            <div className="grid grid-cols-2 gap-3 text-sm">
                              <div>
                                <label className="mb-1 block text-xs font-medium text-gray-500">
                                  Key concepts (one per line)
                                </label>
                                <textarea
                                  className="w-full rounded-md border border-gray-300 p-2 text-sm"
                                  rows={3}
                                  value={topic.keyConcepts.join('\n')}
                                  onChange={(e) =>
                                    updateTopic(unitIdx, topicIdx, {
                                      keyConcepts: linesToList(e.target.value),
                                    })
                                  }
                                />
                              </div>
                              <div>
                                <label className="mb-1 block text-xs font-medium text-gray-500">
                                  Learning outcomes (one per line)
                                </label>
                                <textarea
                                  className="w-full rounded-md border border-gray-300 p-2 text-sm"
                                  rows={3}
                                  value={topic.learningOutcomes.join('\n')}
                                  onChange={(e) =>
                                    updateTopic(unitIdx, topicIdx, {
                                      learningOutcomes: linesToList(e.target.value),
                                    })
                                  }
                                />
                              </div>
                            </div>

                            {topic.clos.length > 0 && (
                              <div className="flex flex-wrap gap-1 pt-1">
                                {topic.clos.map((clo) => (
                                  <span
                                    key={clo.code}
                                    className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600"
                                    title={clo.description}
                                  >
                                    {clo.code}
                                  </span>
                                ))}
                              </div>
                            )}
                          </div>
                        ))}

                        <Button variant="secondary" size="sm" onClick={() => addTopic(unitIdx)}>
                          + Add topic
                        </Button>
                      </CardContent>
                    </Card>
                  ))}
                </div>

                <Button variant="secondary" onClick={addUnit}>
                  + Add unit
                </Button>

                <div className="sticky bottom-4 flex justify-end">
                  <Button size="lg" isLoading={approving} onClick={() => void handleApprove()}>
                    Approve curriculum structure
                  </Button>
                </div>
              </>
            )}

            {job.status === 'pending' || job.status === 'parsing' ? null : !job.parsedStructure &&
              job.status !== 'failed' && <Banner tone="warning">No parsed structure available yet.</Banner>}
          </>
        )}
      </div>
    </AppShell>
  )
}

function StatusBanner({ status, error }: { status: string; error?: string }) {
  if (status === 'failed') {
    return <Banner tone="error">Parsing failed: {error ?? 'Unknown error.'}</Banner>
  }
  if (status === 'approved') {
    return <Banner tone="success">This curriculum has already been approved.</Banner>
  }
  return <Banner tone="info">Status: {status}</Banner>
}
