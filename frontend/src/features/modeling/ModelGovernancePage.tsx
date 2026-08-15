import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, FlaskConical, XCircle } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner, StatusPill } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { listCandidateModelSnapshots, promoteModelSnapshot, rejectModelSnapshot } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { ModelSnapshot } from '@/types/api'

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

const MODEL_TYPE_LABELS: Record<string, string> = {
  bkt_parameters: 'BKT parameters',
  dina_parameters: 'DINA slip/guess',
  gdina_parameters: 'G-DINA parameters',
  irt_calibration: 'IRT item calibration',
  mirt_calibration: 'MIRT calibration',
  dkt_model: 'DKT model',
  prerequisite_graph: 'Prerequisite graph',
  qmatrix: 'Q-matrix',
  fusion_policy: 'Fusion policy',
  recommendation_policy: 'Recommendation policy',
  student_state_snapshot: 'Student state snapshot',
}

// EG-GCKT Milestone 9: review queue for BKT/DINA/IRT parameter refits
// ai-service's nightly refit_worker.py produces (spec section 19's
// candidate/validated/active/rejected governance lifecycle). Nothing here
// is ever auto-activated -- promoting is the only way a refit affects the
// live engines, and the previously-active snapshot survives as
// 'superseded' rather than being deleted, so promoting an older one again
// is how a bad refit gets rolled back.
export function ModelGovernancePage() {
  const queryClient = useQueryClient()
  const [actionError, setActionError] = useState<string | null>(null)

  const candidatesQuery = useQuery({
    queryKey: queryKeys.modelSnapshotCandidates(),
    queryFn: listCandidateModelSnapshots,
  })

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.modelSnapshotCandidates() })

  const promoteMutation = useMutation({
    mutationFn: promoteModelSnapshot,
    onSuccess: () => void invalidate(),
    onError: (err) => setActionError(apiErrorMessage(err, 'Could not promote this snapshot.')),
  })
  const rejectMutation = useMutation({
    mutationFn: rejectModelSnapshot,
    onSuccess: () => void invalidate(),
    onError: (err) => setActionError(apiErrorMessage(err, 'Could not reject this snapshot.')),
  })

  const pending = promoteMutation.isPending || rejectMutation.isPending

  return (
    <AppShell
      title="Model Governance"
      description="Candidate BKT/DINA/IRT parameter refits produced by the nightly recalibration job, pending review before they can affect live learner-state estimates."
    >
      <div className="space-y-6">
        {actionError && <Banner tone="error">{actionError}</Banner>}

        <Card>
          <CardHeader>
            <CardTitle>Pending candidates</CardTitle>
          </CardHeader>
          <CardContent>
            {candidatesQuery.isLoading && (
              <div className="flex items-center gap-2 text-sm text-gray-500">
                <Spinner /> Loading candidate snapshots…
              </div>
            )}
            {candidatesQuery.isError && (
              <Banner tone="error">{apiErrorMessage(candidatesQuery.error, 'Could not load candidate snapshots.')}</Banner>
            )}
            {candidatesQuery.data && candidatesQuery.data.length === 0 && (
              <EmptyState
                icon={FlaskConical}
                title="Nothing pending review"
                description="No candidate parameter refits right now -- either the nightly job hasn't found enough new evidence for any skill/item yet, or every candidate has already been reviewed."
              />
            )}

            {candidatesQuery.data && candidatesQuery.data.length > 0 && (
              <ul className="space-y-4">
                {candidatesQuery.data.map((snapshot) => (
                  <SnapshotRow
                    key={snapshot.id}
                    snapshot={snapshot}
                    disabled={pending}
                    onPromote={() => promoteMutation.mutate(snapshot.id)}
                    onReject={() => rejectMutation.mutate(snapshot.id)}
                  />
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </AppShell>
  )
}

function SnapshotRow({
  snapshot,
  disabled,
  onPromote,
  onReject,
}: {
  snapshot: ModelSnapshot
  disabled: boolean
  onPromote: () => void
  onReject: () => void
}) {
  const summary = snapshot.trainingSummary as { observations?: number; students?: number; logLikelihood?: number; method?: string } | undefined

  return (
    <li className="rounded-lg border border-gray-200 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium text-gray-900">{MODEL_TYPE_LABELS[snapshot.modelType] ?? snapshot.modelType}</span>
            <StatusPill tone="neutral">v{snapshot.version}</StatusPill>
            {snapshot.scope && <StatusPill tone="seal">scope: {snapshot.scope.slice(0, 8)}…</StatusPill>}
          </div>
          <p className="mt-1 text-sm text-gray-500">Produced {formatDate(snapshot.createdAt)}</p>
          {summary && (
            <p className="mt-1 text-sm text-gray-500">
              {summary.observations !== undefined && `${summary.observations} observations`}
              {summary.students !== undefined && ` · ${summary.students} students`}
              {summary.logLikelihood !== undefined && ` · log-likelihood ${summary.logLikelihood}`}
              {summary.method && ` · ${summary.method}`}
            </p>
          )}
          <pre className="mt-2 rounded bg-gray-50 p-2 text-xs text-gray-700">{JSON.stringify(snapshot.config, null, 2)}</pre>
        </div>
        <div className="flex shrink-0 gap-2">
          <Button variant="secondary" disabled={disabled} onClick={onPromote}>
            <CheckCircle2 className="h-4 w-4" aria-hidden />
            Promote
          </Button>
          <Button variant="danger" disabled={disabled} onClick={onReject}>
            <XCircle className="h-4 w-4" aria-hidden />
            Reject
          </Button>
        </div>
      </div>
    </li>
  )
}
