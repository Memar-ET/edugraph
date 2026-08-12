import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { GitBranch, History } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Input, Label, Spinner, StatusPill } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { getSubjectVersions, supersedeSubject } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

function formatDate(iso: string | undefined): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

// Feature 1.3: a mid-year revision is promoted under a NEW subject code
// through the normal upload/approve pipeline (unchanged), then explicitly
// linked here as superseding the old one -- old rows are never mutated,
// so gap_records/study_plans/exam questions pointing at an old version's
// topic_id/clo_code keep meaning exactly what they meant when written.
export function CurriculumVersionsPage() {
  const queryClient = useQueryClient()
  const [lookupCode, setLookupCode] = useState('')
  const [activeCode, setActiveCode] = useState<string | null>(null)

  const [newCode, setNewCode] = useState('')
  const [previousCode, setPreviousCode] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  const versionsQuery = useQuery({
    queryKey: queryKeys.subjectVersions(activeCode ?? ''),
    queryFn: () => getSubjectVersions(activeCode!),
    enabled: Boolean(activeCode),
  })

  const loadLineage = (code?: string) => {
    const target = (code ?? lookupCode).trim().toUpperCase()
    if (!target) return
    setLookupCode(target)
    setActiveCode(target)
  }

  const handleSupersede = async () => {
    const newC = newCode.trim().toUpperCase()
    const prevC = previousCode.trim().toUpperCase()
    if (!newC || !prevC) return
    setSubmitting(true)
    setSubmitError(null)
    try {
      await supersedeSubject(newC, prevC)
      setNewCode('')
      setPreviousCode('')
      loadLineage(newC)
      await queryClient.invalidateQueries({ queryKey: queryKeys.subjectVersions(newC) })
    } catch (err) {
      setSubmitError(apiErrorMessage(err, 'Could not link the new version.'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AppShell
      title="Curriculum Versions"
      description="Mid-year revisions are promoted under a new subject code, then linked here as superseding the old one."
    >
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>View a lineage</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap items-end gap-2">
              <div>
                <Label htmlFor="lookupCode">Subject code</Label>
                <Input
                  id="lookupCode"
                  value={lookupCode}
                  onChange={(e) => setLookupCode(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && loadLineage()}
                  placeholder="BIO-G9"
                  className="w-40"
                />
              </div>
              <Button variant="secondary" onClick={() => loadLineage()}>
                <History className="h-4 w-4" aria-hidden />
                Load history
              </Button>
            </div>

            {versionsQuery.isLoading && (
              <div className="mt-4 flex items-center gap-2 text-sm text-gray-500">
                <Spinner /> Loading version history…
              </div>
            )}
            {versionsQuery.isError && (
              <Banner tone="error" className="mt-4">
                {apiErrorMessage(versionsQuery.error, 'Could not load version history.')}
              </Banner>
            )}
            {versionsQuery.data && versionsQuery.data.length === 0 && (
              <EmptyState
                className="mt-4"
                icon={GitBranch}
                title="No versions found"
                description={`${activeCode} hasn't been approved yet, or has no recorded revision lineage.`}
              />
            )}

            {versionsQuery.data && versionsQuery.data.length > 0 && (
              <ol className="mt-4 space-y-3 border-l-2 border-gray-100 pl-4">
                {versionsQuery.data.map((v) => (
                  <li key={v.code} className="relative">
                    <span className="absolute -left-[21px] top-1.5 h-2.5 w-2.5 rounded-full bg-primary-500" />
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-gray-900">{v.code}</span>
                      <StatusPill tone="neutral">v{v.version}</StatusPill>
                      {v.isCurrent ? (
                        <StatusPill tone="health">Current</StatusPill>
                      ) : (
                        <StatusPill tone="seal">Superseded {formatDate(v.supersededAt)}</StatusPill>
                      )}
                    </div>
                    <p className="text-sm text-gray-500">
                      {v.academicYear} · created {formatDate(v.createdAt)}
                    </p>
                  </li>
                ))}
              </ol>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Link a revision</CardTitle>
            <p className="mt-1 text-sm text-gray-500">
              Upload and approve the revised curriculum under a new subject code first (normal pipeline), then
              link it here as superseding the old one.
            </p>
          </CardHeader>
          <CardContent className="space-y-4">
            {submitError && <Banner tone="error">{submitError}</Banner>}
            <div className="flex flex-wrap items-end gap-2">
              <div>
                <Label htmlFor="newCode">New subject code</Label>
                <Input
                  id="newCode"
                  value={newCode}
                  onChange={(e) => setNewCode(e.target.value)}
                  placeholder="BIO-G9-2027"
                  className="w-44"
                />
              </div>
              <div>
                <Label htmlFor="previousCode">Supersedes</Label>
                <Input
                  id="previousCode"
                  value={previousCode}
                  onChange={(e) => setPreviousCode(e.target.value)}
                  placeholder="BIO-G9"
                  className="w-44"
                />
              </div>
              <Button
                isLoading={submitting}
                disabled={!newCode.trim() || !previousCode.trim()}
                onClick={() => void handleSupersede()}
              >
                <GitBranch className="h-4 w-4" aria-hidden />
                Link as new version
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </AppShell>
  )
}
