import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, Lightbulb, XCircle } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, Button, Card, CardContent, CardHeader, CardTitle, EmptyState, Spinner, StatusPill } from '@components/ui'
import { apiErrorMessage } from '@lib/api/client'
import { listCandidateMisconceptions, reviewMisconception } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { MisconceptionHypothesis } from '@/types/api'

// EG-GCKT Milestone 6 (spec section 11): candidate misconception
// hypotheses the gap-analysis pipeline proposed from a repeated
// wrong-answer pattern on one topic, pending teacher confirm/reject.
// Nothing here is ever auto-confirmed -- a hypothesis only folds into a
// student's skill_states.misconception_state once a teacher confirms it.
export function MisconceptionReviewPage() {
  const queryClient = useQueryClient()

  const candidatesQuery = useQuery({
    queryKey: queryKeys.candidateMisconceptions(),
    queryFn: listCandidateMisconceptions,
  })

  const invalidate = () => queryClient.invalidateQueries({ queryKey: queryKeys.candidateMisconceptions() })

  const reviewMutation = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'confirmed' | 'rejected' }) => reviewMisconception(id, decision),
    onSuccess: () => void invalidate(),
  })

  return (
    <AppShell
      title="Misconception Review"
      description="Candidate misconceptions the AI proposed from repeated wrong-answer patterns -- confirm or reject each before it's shown to anyone as a real finding."
    >
      <Card>
        <CardHeader>
          <CardTitle>Pending candidates</CardTitle>
        </CardHeader>
        <CardContent>
          {candidatesQuery.isLoading && (
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <Spinner /> Loading candidate misconceptions…
            </div>
          )}
          {candidatesQuery.isError && (
            <Banner tone="error">{apiErrorMessage(candidatesQuery.error, 'Could not load candidate misconceptions.')}</Banner>
          )}
          {candidatesQuery.data && candidatesQuery.data.length === 0 && (
            <EmptyState
              icon={Lightbulb}
              title="Nothing pending review"
              description="No candidate misconceptions right now -- either no repeated wrong-answer pattern has been detected recently, or everything found so far has already been reviewed."
            />
          )}
          {candidatesQuery.data && candidatesQuery.data.length > 0 && (
            <ul className="space-y-4">
              {candidatesQuery.data.map((m) => (
                <MisconceptionRow
                  key={m.id}
                  hypothesis={m}
                  disabled={reviewMutation.isPending}
                  onConfirm={() => reviewMutation.mutate({ id: m.id, decision: 'confirmed' })}
                  onReject={() => reviewMutation.mutate({ id: m.id, decision: 'rejected' })}
                />
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </AppShell>
  )
}

function MisconceptionRow({
  hypothesis,
  disabled,
  onConfirm,
  onReject,
}: {
  hypothesis: MisconceptionHypothesis
  disabled: boolean
  onConfirm: () => void
  onReject: () => void
}) {
  return (
    <li className="rounded-lg border border-gray-200 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium text-gray-900">{hypothesis.topicTitle}</span>
            {hypothesis.confidence !== undefined && (
              <StatusPill tone="neutral">{Math.round(hypothesis.confidence * 100)}% confidence</StatusPill>
            )}
            {hypothesis.generatedByModel && <StatusPill tone="seal">{hypothesis.generatedByModel}</StatusPill>}
          </div>
          <p className="mt-2 text-sm text-gray-900">{hypothesis.misconceptionText}</p>
          {hypothesis.triggerPattern && (
            <p className="mt-1 text-sm text-gray-500">Trigger pattern: {hypothesis.triggerPattern}</p>
          )}
          {hypothesis.interventionText && (
            <p className="mt-1 text-sm text-gray-500">Suggested fix: {hypothesis.interventionText}</p>
          )}
        </div>
        <div className="flex shrink-0 gap-2">
          <Button variant="secondary" disabled={disabled} onClick={onConfirm}>
            <CheckCircle2 className="h-4 w-4" aria-hidden />
            Confirm
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
