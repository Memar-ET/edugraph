import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'

import { AppShell } from '@components/layout'
import { Banner, Card, CardContent, EmptyState, Spinner } from '@components/ui'
import { getMySubjectProfiles } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { cn } from '@lib/utils/cn'
import { useAuthStore } from '@stores/auth.store'
import type { SubjectProfile } from '@/types/api'


function masteryColor(pct: number) {
  if (pct >= 75) return 'text-health-700'
  if (pct >= 50) return 'text-seal-700'
  return 'text-alert-700'
}

function masteryBg(pct: number) {
  if (pct >= 75) return 'bg-health-50 border-health-200'
  if (pct >= 50) return 'bg-seal-50 border-seal-200'
  return 'bg-alert-50 border-alert-200'
}

function masteryLabel(pct: number) {
  if (pct >= 75) return 'On track'
  if (pct >= 50) return 'Developing'
  return 'Needs focus'
}

export function SubjectProfilePage() {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.mySubjectProfiles(),
    queryFn: getMySubjectProfiles,
  })

  return (
    <AppShell>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Subject Profiles</h1>
          <p className="mt-1 text-sm text-gray-500">
            Your mastery and gap summary for each subject you have been assessed on.
          </p>
        </div>

        {isError && (
          <Banner
            tone="error"
            title="Failed to load subject profiles"
            description="Try refreshing the page."
          />
        )}

        {isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : !data?.length ? (
          <EmptyState
            title="No subject profiles yet"
            description="Complete an exam to see your subject health profiles here."
          />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {data.map((profile) => (
              <SubjectCard
                key={profile.subjectCode}
                profile={profile}
                studentId={user?.id}
                onExplain={(topicId) => {
                  if (!user?.id) return
                  navigate({
                    to: '/students/$studentId/topics/$topicId/explain',
                    params: { studentId: user.id, topicId },
                  })
                }}
              />
            ))}
          </div>
        )}
      </div>
    </AppShell>
  )
}

function SubjectCard({
  profile,
}: {
  profile: SubjectProfile
  studentId?: string
  onExplain?: (topicId: string) => void
}) {
  const pct = Math.round(profile.currentMasteryPct * 100)
  const label = masteryLabel(pct)
  const color = masteryColor(pct)
  const bg = masteryBg(pct)

  return (
    <Card className="flex flex-col">
      <CardContent className="flex flex-1 flex-col gap-4 py-5">
        {/* Header */}
        <div className="flex items-start justify-between gap-2">
          <div>
            <p className="font-semibold text-gray-900">{profile.subjectName}</p>
            <p className="text-xs text-gray-500">
              Grade {profile.gradeLevel} · {profile.examsAnalyzed} exam
              {profile.examsAnalyzed !== 1 ? 's' : ''} analysed
            </p>
          </div>
          <span className={cn('rounded-full border px-2 py-0.5 text-xs font-medium', bg, color)}>
            {label}
          </span>
        </div>

        {/* Mastery bar */}
        <div>
          <div className="mb-1 flex items-center justify-between text-sm">
            <span className="text-gray-600">Overall mastery</span>
            <span className={cn('font-bold', color)}>{pct}%</span>
          </div>
          <div className="h-2 w-full overflow-hidden rounded-full bg-gray-100">
            <div
              className={cn(
                'h-full rounded-full transition-all',
                pct >= 75 ? 'bg-health-500' : pct >= 50 ? 'bg-seal-500' : 'bg-alert-500',
              )}
              style={{ width: `${pct}%` }}
              role="progressbar"
              aria-valuenow={pct}
              aria-valuemin={0}
              aria-valuemax={100}
            />
          </div>
        </div>

        {/* Last updated */}
        <p className="text-xs text-gray-400">
          Updated {new Date(profile.lastUpdated).toLocaleDateString()}
        </p>
      </CardContent>
    </Card>
  )
}
