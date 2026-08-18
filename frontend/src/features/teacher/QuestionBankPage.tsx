import { useState } from 'react'
import { Filter, Search } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'

import { AppShell } from '@components/layout'
import { Banner, EmptyState, Spinner } from '@components/ui'
import { listExams, listExamQuestions } from '@lib/api/endpoints'
import { queryKeys } from '@lib/query/keys'
import type { ExamQuestion } from '@/types/api'

const TYPE_COLORS: Record<string, string> = {
  mcq: 'bg-blue-100 text-blue-800',
  short_answer: 'bg-emerald-100 text-emerald-800',
  long_answer: 'bg-amber-100 text-amber-800',
  essay: 'bg-violet-100 text-violet-800',
  calculation: 'bg-rose-100 text-rose-800',
}

export function QuestionBankPage() {
  const [selectedExamId, setSelectedExamId] = useState('')
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState('all')

  const { data: examsData, isLoading: examsLoading } = useQuery({
    queryKey: queryKeys.exams(),
    queryFn: () => listExams(1, 100),
  })

  const { data: questions, isLoading: questionsLoading, isError, error } = useQuery({
    queryKey: queryKeys.examQuestions(selectedExamId),
    queryFn: () => listExamQuestions(selectedExamId),
    enabled: !!selectedExamId,
  })

  const exams = examsData?.items ?? []

  const filtered = (questions ?? []).filter((q: ExamQuestion) => {
    const matchesSearch =
      search === '' || q.questionText.toLowerCase().includes(search.toLowerCase())
    const matchesType = typeFilter === 'all' || q.questionType === typeFilter
    return matchesSearch && matchesType
  })

  return (
    <AppShell
      title="Question Bank Repository"
      description="Browse assessment items linked to CLO standards. Select an exam to inspect its questions."
    >
      <div className="space-y-4">
        {/* Exam selector + filters */}
        <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm sm:flex-row sm:items-center">
          {examsLoading ? (
            <Spinner />
          ) : (
            <select
              value={selectedExamId}
              onChange={(e) => setSelectedExamId(e.target.value)}
              className="flex-1 rounded-lg border border-slate-200 bg-slate-50 px-3 py-1.5 text-xs text-slate-800 focus:border-teal-500 focus:outline-none"
            >
              <option value="">— Select an exam —</option>
              {exams.map((e) => (
                <option key={e.examId} value={e.examId}>
                  {e.title} ({e.status})
                </option>
              ))}
            </select>
          )}
          {selectedExamId && (
            <>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400" />
                <input
                  type="text"
                  placeholder="Search question text…"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="rounded-lg border border-slate-200 py-1.5 pl-8 pr-3 text-xs text-slate-800 focus:border-teal-500 focus:outline-none"
                />
              </div>
              <div className="flex items-center gap-1.5 text-xs text-slate-600">
                <Filter className="h-3.5 w-3.5 text-slate-400" />
                <select
                  value={typeFilter}
                  onChange={(e) => setTypeFilter(e.target.value)}
                  className="rounded-md border border-slate-200 bg-slate-50 px-2 py-1 text-xs text-slate-700 focus:outline-none"
                >
                  <option value="all">All Types</option>
                  <option value="mcq">MCQ</option>
                  <option value="short_answer">Short Answer</option>
                  <option value="long_answer">Long Answer</option>
                  <option value="essay">Essay</option>
                  <option value="calculation">Calculation</option>
                </select>
              </div>
            </>
          )}
        </div>

        {!selectedExamId && (
          <EmptyState
            title="Select an exam"
            description="Choose an exam above to browse its questions and CLO mappings."
          />
        )}

        {selectedExamId && questionsLoading && (
          <div className="flex items-center gap-2 py-8 text-xs text-slate-500">
            <Spinner /> Loading questions…
          </div>
        )}

        {selectedExamId && isError && (
          <Banner tone="error">{(error as Error).message ?? 'Could not load questions.'}</Banner>
        )}

        {selectedExamId && !questionsLoading && filtered.length === 0 && (
          <EmptyState title="No questions found" description="Adjust your search or upload an answer key to associate CLOs." />
        )}

        {filtered.length > 0 && (
          <>
            <p className="text-xs text-slate-500">{filtered.length} question{filtered.length !== 1 ? 's' : ''}</p>
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              {filtered.map((q: ExamQuestion) => (
                <div key={q.id} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm hover:border-slate-300 transition-all">
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-xs font-bold text-teal-700">
                      Q{q.sequenceNumber}{q.partLabel ? ` (${q.partLabel})` : ''}
                    </span>
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${TYPE_COLORS[q.questionType] ?? 'bg-slate-100 text-slate-700'}`}>
                      {q.questionType.replace(/_/g, ' ')}
                    </span>
                  </div>

                  <p className="mt-2 text-xs font-medium text-slate-900 leading-relaxed">{q.questionText}</p>

                  <div className="mt-3 flex items-center justify-between text-[11px] text-slate-500 pt-3 border-t border-slate-100">
                    <span>Marks: <strong className="text-slate-900">{q.marks}</strong></span>
                    <Link
                      to="/teacher/exams/$examId/quality"
                      params={{ examId: selectedExamId }}
                      className="font-semibold text-teal-700 hover:underline"
                    >
                      Quality Report
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </AppShell>
  )
}
