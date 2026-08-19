import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, XCircle, Brain, Loader2 } from 'lucide-react'
import { AppShell } from '@components/layout'
import { listCandidateMisconceptions, reviewMisconception } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function TeacherMisconceptionReviewPage() {
  const queryClient = useQueryClient()

  const { data: items = [], isLoading, isError, error } = useQuery({
    queryKey: queryKeys.candidateMisconceptions(),
    queryFn: listCandidateMisconceptions,
  })

  const reviewMutation = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'confirmed' | 'rejected' }) =>
      reviewMisconception(id, decision),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.candidateMisconceptions() }),
  })

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="font-display font-extrabold text-2xl text-slate-900 tracking-tight">
            AI Misconception Hypothesis Desk
          </h1>
          <p className="text-xs text-slate-500 mt-1">
            Review AI-detected conceptual errors extracted from student exam responses across your sections.
          </p>
        </div>

        {isError && (
          <div className="rounded-2xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
            Failed to load misconceptions: {(error as Error).message}
          </div>
        )}

        {isLoading ? (
          <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="h-28 animate-pulse rounded-2xl bg-slate-100" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="rounded-2xl border border-slate-200 bg-white p-12 text-center shadow-sm">
            <Brain className="mx-auto h-10 w-10 text-slate-300 mb-3" />
            <p className="text-sm font-bold text-slate-700">No pending misconceptions</p>
            <p className="text-xs text-slate-400 mt-1">
              AI-generated hypotheses appear here when students show consistent error patterns.
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {items.map((item) => (
              <div
                key={item.id}
                className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm space-y-3"
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-mono text-xs font-bold text-teal-700">
                        {item.id.slice(0, 8)}…
                      </span>
                      <span className="rounded bg-slate-100 px-2 py-0.5 text-[10px] font-bold text-slate-700">
                        {item.status}
                      </span>
                    </div>
                    <h3 className="font-bold text-sm text-slate-900 mt-1 leading-snug">
                      {item.misconceptionText}
                    </h3>
                    {item.triggerPattern && (
                      <p className="text-xs text-slate-500 mt-1 italic">
                        Trigger: {item.triggerPattern}
                      </p>
                    )}
                  </div>
                  <span
                    className={`ml-3 shrink-0 rounded-full px-2.5 py-0.5 text-[10px] font-bold uppercase ${
                      item.status === 'confirmed'
                        ? 'bg-emerald-100 text-emerald-800'
                        : item.status === 'rejected'
                        ? 'bg-rose-100 text-rose-800'
                        : 'bg-amber-100 text-amber-800'
                    }`}
                  >
                    {item.status}
                  </span>
                </div>

                <div className="flex items-center justify-between text-xs text-slate-500 pt-2 border-t border-slate-100">
                  <span>
                    Confidence:{' '}
                    <strong className="text-teal-700">{Math.round((item.confidence ?? 0) * 100)}%</strong>
                  </span>
                  {item.interventionText && (
                    <span className="text-slate-400 italic truncate max-w-[60%]">
                      Intervention: {item.interventionText}
                    </span>
                  )}
                </div>

                {item.status === 'candidate' && (
                  <div className="flex items-center justify-end gap-2 pt-2">
                    <button
                      type="button"
                      disabled={reviewMutation.isPending}
                      onClick={() => reviewMutation.mutate({ id: item.id, decision: 'rejected' })}
                      className="flex items-center gap-1 rounded-xl border border-slate-200 px-3 py-1.5 text-xs font-semibold text-rose-700 hover:bg-rose-50 disabled:opacity-50"
                    >
                      {reviewMutation.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <XCircle className="h-4 w-4" />
                      )}
                      Reject
                    </button>
                    <button
                      type="button"
                      disabled={reviewMutation.isPending}
                      onClick={() => reviewMutation.mutate({ id: item.id, decision: 'confirmed' })}
                      className="flex items-center gap-1 rounded-xl bg-teal-700 px-4 py-1.5 text-xs font-semibold text-white hover:bg-teal-800 disabled:opacity-50"
                    >
                      {reviewMutation.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <CheckCircle2 className="h-4 w-4" />
                      )}
                      Confirm &amp; Assign Remediation
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </AppShell>
  )
}
