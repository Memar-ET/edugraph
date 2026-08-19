import { Flame, Star, Trophy, BookOpen, ClipboardList, Layers, Lock } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { AppShell } from '@components/layout'
import { getMySubjectProfiles, listMyStudyPlans } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { SubjectProfile, StudyPlan } from '@/types/api'

interface Achievement {
  id: string
  title: string
  category: string
  description: string
  unlocked: boolean
  unlockedDate?: string
}

function deriveAchievements(
  profiles: SubjectProfile[],
  plans: StudyPlan[],
): Achievement[] {
  const achievements: Achievement[] = []

  for (const p of profiles) {
    if (p.currentMasteryPct >= 85) {
      achievements.push({
        id: `mastery-${p.subjectCode}`,
        title: `${p.subjectName} Expert`,
        category: 'Subject Mastery',
        description: `Achieved ${Math.round(p.currentMasteryPct)}% mastery in ${p.subjectName} Grade ${p.gradeLevel}.`,
        unlocked: true,
        unlockedDate: new Date(p.lastUpdated).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' }),
      })
    } else if (p.currentMasteryPct >= 60) {
      achievements.push({
        id: `progress-${p.subjectCode}`,
        title: `${p.subjectName} Progress`,
        category: 'Subject Mastery',
        description: `Reached 60%+ mastery in ${p.subjectName} Grade ${p.gradeLevel}.`,
        unlocked: true,
        unlockedDate: new Date(p.lastUpdated).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' }),
      })
    } else {
      achievements.push({
        id: `unlock-${p.subjectCode}`,
        title: `${p.subjectName} Proficient`,
        category: 'Subject Mastery',
        description: `Reach 85% mastery in ${p.subjectName} Grade ${p.gradeLevel} to unlock.`,
        unlocked: false,
      })
    }
  }

  if (plans.length > 0) {
    achievements.push({
      id: 'study-plan',
      title: 'Study Planner',
      category: 'Learning Habits',
      description: 'Generated your first AI-powered personalized study plan.',
      unlocked: true,
      unlockedDate: new Date(plans[0]!.generatedAt).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' }),
    })
  } else {
    achievements.push({
      id: 'study-plan',
      title: 'Study Planner',
      category: 'Learning Habits',
      description: 'Generate your first AI-powered study plan to unlock.',
      unlocked: false,
    })
  }

  const totalExams = profiles.reduce((sum, p) => sum + p.examsAnalyzed, 0)
  if (totalExams >= 5) {
    achievements.push({
      id: 'exam-5',
      title: 'Exam Veteran',
      category: 'Assessment',
      description: `Completed ${totalExams} exam analysis sessions with the AI engine.`,
      unlocked: true,
    })
  } else {
    achievements.push({
      id: 'exam-5',
      title: 'Exam Veteran',
      category: 'Assessment',
      description: 'Complete 5 analyzed exams to unlock.',
      unlocked: false,
    })
  }

  return achievements
}

export function StudentAchievementsPage() {
  const { data: profiles, isLoading: loadingProfiles } = useQuery({
    queryKey: queryKeys.mySubjectProfiles(),
    queryFn: getMySubjectProfiles,
  })
  const { data: plans, isLoading: loadingPlans } = useQuery({
    queryKey: queryKeys.myStudyPlans(),
    queryFn: listMyStudyPlans,
  })

  const isLoading = loadingProfiles || loadingPlans
  const achievements = deriveAchievements(profiles ?? [], plans ?? [])
  const unlocked = achievements.filter((a) => a.unlocked)
  const totalExams = (profiles ?? []).reduce((sum, p) => sum + p.examsAnalyzed, 0)

  return (
    <AppShell
      title="Mastery Badges & Achievements"
      description="Track your topic milestones and proficiency badges."
    >
      <div className="space-y-4">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-50 text-amber-500">
              <Flame className="h-6 w-6 fill-amber-500" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Exams Analyzed</p>
              <p className="text-2xl font-bold text-slate-900 font-display">
                {isLoading ? '—' : totalExams}
              </p>
            </div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-teal-50 text-teal-600">
              <Trophy className="h-6 w-6" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Badges Earned</p>
              <p className="text-2xl font-bold text-teal-700 font-display">
                {isLoading ? '—' : `${unlocked.length} / ${achievements.length}`}
              </p>
            </div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600">
              <Star className="h-6 w-6 fill-emerald-500" />
            </div>
            <div>
              <p className="text-xs font-medium text-slate-500">Subjects Tracked</p>
              <p className="text-2xl font-bold text-slate-900 font-display">
                {isLoading ? '—' : profiles?.length ?? 0}
              </p>
            </div>
          </div>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4 animate-pulse">
            {[0, 1, 2, 3].map((i) => <div key={i} className="h-40 bg-slate-200 rounded-xl" />)}
          </div>
        ) : achievements.length === 0 ? (
          <div className="rounded-2xl border border-slate-200 bg-white p-10 text-center shadow-sm">
            <Layers className="mx-auto h-10 w-10 text-slate-300 mb-3" />
            <p className="font-semibold text-slate-700">No achievements yet</p>
            <p className="text-xs text-slate-500 mt-1">Take exams and generate study plans to earn your first badges.</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {achievements.map((a) => {
              const Icon = a.category === 'Subject Mastery' ? BookOpen
                : a.category === 'Assessment' ? ClipboardList
                : Flame
              return (
                <div
                  key={a.id}
                  className={`rounded-xl border p-5 transition-all ${
                    a.unlocked
                      ? 'border-slate-200 bg-white shadow-sm'
                      : 'border-slate-200 bg-slate-50/60 opacity-60'
                  }`}
                >
                  <div className="mb-3 flex h-10 w-10 items-center justify-center rounded-xl bg-teal-50">
                    {a.unlocked
                      ? <Icon className="h-5 w-5 text-teal-600" />
                      : <Lock className="h-5 w-5 text-slate-400" />}
                  </div>
                  <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400">{a.category}</span>
                  <h4 className="font-bold text-sm text-slate-900 mt-0.5">{a.title}</h4>
                  <p className="mt-1 text-xs text-slate-500 leading-relaxed">{a.description}</p>
                  {a.unlocked && a.unlockedDate && (
                    <p className="mt-3 text-[10px] font-semibold text-emerald-600">Unlocked {a.unlockedDate}</p>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </AppShell>
  )
}
