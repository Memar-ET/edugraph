import { Lightbulb, BookOpen } from 'lucide-react'
import { useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { AppShell } from '@components/layout'
import { getMySubjectProfiles } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'

export function StudentMisconceptionJournalPage() {
  const navigate = useNavigate()
  const { data: profiles, isLoading, error } = useQuery({
    queryKey: queryKeys.mySubjectProfiles(),
    queryFn: getMySubjectProfiles,
  })

  const weakAreas = profiles?.filter((p) => p.currentMasteryPct < 60) ?? []

  if (isLoading) {
    return (
      <AppShell title="Misconception Diagnostic Journal">
        <div className="space-y-4 animate-pulse">
          {[0, 1, 2].map((i) => <div key={i} className="h-32 bg-slate-200 rounded-xl" />)}
        </div>
      </AppShell>
    )
  }

  if (error) {
    return (
      <AppShell title="Misconception Diagnostic Journal">
        <div className="rounded-2xl border border-rose-200 bg-rose-50 p-5 text-rose-800 text-sm">
          Failed to load data: {(error as Error).message}
        </div>
      </AppShell>
    )
  }

  return (
    <AppShell
      title="Misconception Diagnostic Journal"
      description="Personalized diagnostic analysis of conceptual misconceptions flagged from your exam attempts."
    >
      <div className="space-y-4">
        {/* Explanation banner */}
        <div className="rounded-2xl border border-teal-100 bg-teal-50 p-4 text-xs text-teal-800">
          <div className="flex items-center gap-2 font-bold mb-1">
            <Lightbulb className="h-4 w-4 text-teal-600" />
            How this works
          </div>
          <p>
            When your teacher reviews AI-detected misconception hypotheses, confirmed ones will appear here with
            remediation guidance. In the meantime, your subjects with low mastery are listed below as focus areas.
          </p>
        </div>

        {weakAreas.length === 0 && (profiles?.length ?? 0) > 0 ? (
          <div className="rounded-2xl border border-emerald-100 bg-emerald-50 p-8 text-center">
            <Lightbulb className="mx-auto h-8 w-8 text-emerald-400 mb-2" />
            <p className="font-semibold text-emerald-800">Great work — no weak areas detected!</p>
            <p className="text-xs text-emerald-600 mt-1">All your subjects are above the 60% mastery threshold.</p>
          </div>
        ) : weakAreas.length === 0 ? (
          <div className="rounded-2xl border border-slate-200 bg-white p-10 text-center shadow-sm">
            <BookOpen className="mx-auto h-10 w-10 text-slate-300 mb-3" />
            <p className="font-semibold text-slate-700">No data yet</p>
            <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto">
              Take exams to generate your personal misconception analysis. Your weak areas will appear here after your teacher reviews AI-detected patterns.
            </p>
            <button
              type="button"
              onClick={() => void navigate({ to: '/student/exams' })}
              className="mt-4 inline-flex items-center gap-1.5 rounded-xl bg-teal-700 px-4 py-2 text-xs font-semibold text-white hover:bg-teal-800"
            >
              View Available Exams
            </button>
          </div>
        ) : (
          <>
            <h2 className="text-sm font-bold text-slate-900">Subjects Needing Attention</h2>
            {weakAreas.map((profile) => (
              <div key={profile.subjectCode} className="rounded-xl border border-amber-200 bg-white p-5 shadow-sm">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Lightbulb className="h-4 w-4 text-amber-500" />
                    <span className="font-mono text-xs font-bold text-teal-700">{profile.subjectCode}</span>
                    <span className="text-xs font-semibold text-slate-700">
                      {profile.subjectName} · Grade {profile.gradeLevel}
                    </span>
                  </div>
                  <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-[10px] font-bold text-amber-800">
                    Needs Review
                  </span>
                </div>

                <div className="mt-3">
                  <div className="flex items-center justify-between text-xs mb-1">
                    <span className="text-slate-500">Current Mastery</span>
                    <span className="font-bold text-slate-900">{Math.round(profile.currentMasteryPct)}%</span>
                  </div>
                  <div className="h-2 w-full overflow-hidden rounded-full bg-slate-100">
                    <div
                      className="h-full rounded-full bg-amber-500"
                      style={{ width: `${profile.currentMasteryPct}%` }}
                    />
                  </div>
                </div>

                <p className="mt-3 text-xs text-slate-500">
                  Based on {profile.examsAnalyzed} analyzed exam{profile.examsAnalyzed !== 1 ? 's' : ''}.
                  Your teacher will review AI-detected misconceptions for this subject.
                </p>
              </div>
            ))}
          </>
        )}
      </div>
    </AppShell>
  )
}
