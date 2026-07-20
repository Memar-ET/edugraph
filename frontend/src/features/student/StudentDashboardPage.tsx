import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Briefcase, ClipboardList, MessageCircleQuestion, RefreshCw, Sparkles } from 'lucide-react'

import { AppShell } from '@components/layout'
import {
  Banner,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  EmptyState,
  Spinner,
  StatusPill,
  toneForPct,
} from '@components/ui'
import { ScoreGauge } from '@components/charts'
import { apiErrorMessage } from '@lib/api/client'
import {
  generateCareerMatches,
  generateStudyPlan,
  getCareerMatches,
  getMySubjectProfiles,
  listMyStudyPlans,
} from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import { extractLabels, formatPercent } from '@lib/utils/format'
import { formatDate, formatRelative } from '@lib/utils/date'
import { useAuthStore } from '@stores/auth.store'

import { useMyStudentRecord } from './useMyStudentRecord'

export function StudentDashboardPage() {
  const user = useAuthStore((s) => s.user)
  const firstName = user?.full_name.split(' ')[0] ?? 'there'

  return (
    <AppShell title={`Welcome back, ${firstName}`} description="Your subjects, study plan, and next steps.">
      <div className="space-y-8">
        <SubjectHealthSection />
        <div className="grid gap-6 lg:grid-cols-2">
          <StudyPlanSection />
          <CareerMatchesSection />
        </div>
        <QuickLinks />
      </div>
    </AppShell>
  )
}

function SubjectHealthSection() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.mySubjectProfiles(),
    queryFn: getMySubjectProfiles,
  })

  return (
    <section>
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-gray-500">Subject health</h2>
      {isLoading && (
        <div className="flex items-center gap-2 text-sm text-gray-500">
          <Spinner /> Loading your subjects…
        </div>
      )}
      {isError && <Banner tone="error">{apiErrorMessage(error, 'Could not load your subject health.')}</Banner>}
      {!isLoading && !isError && (!data || data.length === 0) && (
        <EmptyState
          icon={Sparkles}
          title="No subject data yet"
          description="Take a published exam and your subject-by-subject mastery will build up here automatically."
        />
      )}
      {data && data.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {data.map((profile) => {
            const weakAreas = extractLabels(profile.topWeakAreas)
            return (
              <Card key={`${profile.subjectCode}-${profile.gradeLevel}`}>
                <CardContent className="flex items-start gap-4 pt-6">
                  <ScoreGauge
                    value={profile.currentMasteryPct}
                    tone={toneForPct(profile.currentMasteryPct) === 'alert' ? 'alert' : toneForPct(profile.currentMasteryPct) === 'seal' ? 'seal' : 'health'}
                    size={72}
                  />
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-display text-base font-semibold text-gray-900">
                      {profile.subjectName}
                    </p>
                    <p className="text-xs text-gray-500">
                      Grade {profile.gradeLevel} · {profile.examsAnalyzed} exam
                      {profile.examsAnalyzed === 1 ? '' : 's'} analyzed
                    </p>
                    {weakAreas.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1">
                        {weakAreas.slice(0, 3).map((area) => (
                          <StatusPill key={area} tone="alert">
                            {area}
                          </StatusPill>
                        ))}
                      </div>
                    )}
                    <p className="mt-2 text-xs text-gray-400">Updated {formatRelative(profile.lastUpdated)}</p>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </section>
  )
}

function StudyPlanSection() {
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.myStudyPlans(),
    queryFn: listMyStudyPlans,
  })

  const generate = useMutation({
    mutationFn: () => generateStudyPlan({}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.myStudyPlans() })
      setTimeout(() => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.myStudyPlans() })
      }, 6000)
    },
  })

  const latest = data?.[0]

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Study plan</CardTitle>
        <Button
          size="sm"
          variant="secondary"
          isLoading={generate.isPending}
          onClick={() => generate.mutate()}
        >
          <RefreshCw className="h-4 w-4" aria-hidden />
          Generate
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        {generate.isSuccess && (
          <Banner tone="info">
            Building your plan from your recent gaps. This can take a moment — refresh if it doesn&apos;t
            appear below.
          </Banner>
        )}
        {generate.isError && (
          <Banner tone="error">{apiErrorMessage(generate.error, 'Could not start a study plan.')}</Banner>
        )}
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Spinner /> Loading…
          </div>
        )}
        {!isLoading && !latest && (
          <EmptyState
            icon={Sparkles}
            title="No study plan yet"
            description="Generate one from your current gaps -- it lays out day-by-day topics to review, prioritizing root causes."
          />
        )}
        {latest && (
          <div>
            <p className="text-sm text-gray-600">
              {latest.totalDays} day{latest.totalDays === 1 ? '' : 's'} · {latest.totalHours}h total ·
              generated {formatDate(latest.generatedAt)}
            </p>
            {latest.planData?.summary && (
              <p className="mt-2 text-sm text-gray-700">{latest.planData.summary}</p>
            )}
            <ul className="mt-3 space-y-2">
              {latest.planData?.days?.slice(0, 3).map((day) => (
                <li key={String(day.day)} className="rounded-md border border-gray-200 px-3 py-2 text-sm">
                  <span className="font-medium text-gray-900">Day {day.day}</span>
                  <ul className="mt-1 list-inside list-disc text-gray-600">
                    {day.blocks?.map((block, i) => (
                      <li key={i}>
                        {block.title} {block.hours ? `(${block.hours}h)` : ''}
                        {block.isRootCause && (
                          <StatusPill tone="alert" className="ml-2">
                            root cause
                          </StatusPill>
                        )}
                      </li>
                    ))}
                  </ul>
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function CareerMatchesSection() {
  const { record, isLoading: isLoadingRecord } = useMyStudentRecord()
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: queryKeys.careerMatches(record?.id ?? 'unknown'),
    queryFn: () => getCareerMatches(record!.id),
    enabled: Boolean(record),
  })

  const generate = useMutation({
    mutationFn: () => generateCareerMatches(record!.id),
    onSuccess: (matches) => {
      queryClient.setQueryData(queryKeys.careerMatches(record!.id), matches)
    },
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Career matches</CardTitle>
        <Button
          size="sm"
          variant="secondary"
          isLoading={generate.isPending}
          disabled={!record}
          onClick={() => generate.mutate()}
        >
          <Briefcase className="h-4 w-4" aria-hidden />
          Find matches
        </Button>
      </CardHeader>
      <CardContent>
        {generate.isError && (
          <Banner tone="error">{apiErrorMessage(generate.error, 'Could not generate career matches.')}</Banner>
        )}
        {(isLoadingRecord || isLoading) && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <Spinner /> Loading…
          </div>
        )}
        {!isLoadingRecord && !isLoading && (!data || data.length === 0) && (
          <EmptyState
            icon={Briefcase}
            title="No matches yet"
            description="Generate career-path matches based on your subject strengths."
          />
        )}
        {data && data.length > 0 && (
          <ul className="space-y-2">
            {data
              .slice()
              .sort((a, b) => b.score - a.score)
              .map((match) => (
                <li
                  key={match.career_path_id}
                  className="flex items-center justify-between rounded-md border border-gray-200 px-3 py-2 text-sm"
                >
                  <span className="font-medium text-gray-900">{match.title}</span>
                  <StatusPill tone={toneForPct(match.score * 100)}>{formatPercent(match.score, { fromRatio: true })} fit</StatusPill>
                </li>
              ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}

function QuickLinks() {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <Link to="/student/exams" className="group">
        <Card className="transition-colors group-hover:border-primary-300">
          <CardContent className="flex items-center gap-3 pt-6">
            <ClipboardList className="h-5 w-5 text-primary-700" aria-hidden />
            <div>
              <p className="font-medium text-gray-900">Take an exam</p>
              <p className="text-sm text-gray-500">Enter the link or ID your teacher shared.</p>
            </div>
          </CardContent>
        </Card>
      </Link>
      <Link to="/student/tutor" className="group">
        <Card className="transition-colors group-hover:border-primary-300">
          <CardContent className="flex items-center gap-3 pt-6">
            <MessageCircleQuestion className="h-5 w-5 text-primary-700" aria-hidden />
            <div>
              <p className="font-medium text-gray-900">Ask the tutor</p>
              <p className="text-sm text-gray-500">Get a personalized explanation based on your gaps.</p>
            </div>
          </CardContent>
        </Card>
      </Link>
    </div>
  )
}
