import { useState } from 'react'
import {
  ClipboardList,
  CheckCircle2,
  ChevronRight,
  ChevronLeft,
  X,
  FileText,
  Sparkles,
} from 'lucide-react'

export function ExamWizard({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [step, setStep] = useState(1)
  const [title, setTitle] = useState('Physics Grade 11 Mid-Term Exam')
  const [subject, setSubject] = useState('Physics')
  const [grade, setGrade] = useState(11)

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-2xl rounded-2xl border border-slate-200 bg-white shadow-2xl overflow-hidden animate-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-100 px-6 py-4 bg-slate-50">
          <div className="flex items-center gap-2">
            <ClipboardList className="h-5 w-5 text-teal-700" />
            <h3 className="font-bold text-sm text-slate-900">Create & Author Assessment Exam</h3>
          </div>
          <button type="button" onClick={onClose} className="rounded-lg p-1 text-slate-400 hover:bg-slate-200">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Step Indicator */}
        <div className="grid grid-cols-4 border-b border-slate-100 bg-slate-50/50 text-[11px] font-semibold text-slate-500 text-center">
          <div className={`py-2.5 ${step === 1 ? 'border-b-2 border-teal-700 text-teal-800 font-bold bg-white' : ''}`}>1. Metadata</div>
          <div className={`py-2.5 ${step === 2 ? 'border-b-2 border-teal-700 text-teal-800 font-bold bg-white' : ''}`}>2. Questions</div>
          <div className={`py-2.5 ${step === 3 ? 'border-b-2 border-teal-700 text-teal-800 font-bold bg-white' : ''}`}>3. CLO Mapping</div>
          <div className={`py-2.5 ${step === 4 ? 'border-b-2 border-teal-700 text-teal-800 font-bold bg-white' : ''}`}>4. Publish</div>
        </div>

        {/* Step Content */}
        <div className="p-6">
          {step === 1 && (
            <div className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-slate-700">Exam Title</label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  className="mt-1 w-full rounded-xl border border-slate-200 p-2.5 text-xs text-slate-900 focus:border-teal-500 focus:outline-none"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs font-semibold text-slate-700">Subject</label>
                  <select
                    value={subject}
                    onChange={(e) => setSubject(e.target.value)}
                    className="mt-1 w-full rounded-xl border border-slate-200 p-2.5 text-xs text-slate-900 focus:outline-none"
                  >
                    <option value="Physics">Physics</option>
                    <option value="Chemistry">Chemistry</option>
                    <option value="Mathematics">Mathematics</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs font-semibold text-slate-700">Grade Level</label>
                  <select
                    value={grade}
                    onChange={(e) => setGrade(Number(e.target.value))}
                    className="mt-1 w-full rounded-xl border border-slate-200 p-2.5 text-xs text-slate-900 focus:outline-none"
                  >
                    <option value={11}>Grade 11</option>
                    <option value={12}>Grade 12</option>
                  </select>
                </div>
              </div>
            </div>
          )}

          {step === 2 && (
            <div className="space-y-3 text-xs text-slate-600">
              <p className="font-semibold text-slate-900">Select Questions from Repository or Upload Paper</p>
              <div className="rounded-xl border border-dashed border-slate-300 p-6 text-center bg-slate-50">
                <FileText className="h-8 w-8 text-slate-400 mx-auto mb-2" />
                <p className="font-bold text-slate-800">Drag and drop exam PDF or DOCX file</p>
                <p className="text-[11px] text-slate-500">or pick questions from Question Bank (24 questions selected)</p>
              </div>
            </div>
          )}

          {step === 3 && (
            <div className="space-y-3 text-xs">
              <div className="flex items-center gap-2 rounded-xl bg-teal-50 p-3 text-teal-800 font-semibold">
                <Sparkles className="h-4 w-4" />
                <span>AI automatic CLO alignment confidence: 94%</span>
              </div>
              <p className="text-slate-600">All 24 questions successfully mapped to Grade 11 Physics CLO standards.</p>
            </div>
          )}

          {step === 4 && (
            <div className="space-y-4 text-xs text-slate-600">
              <div className="rounded-xl border border-slate-200 p-4 bg-slate-50 space-y-1">
                <p className="font-bold text-slate-900 text-sm">{title}</p>
                <p>{subject} (Grade {grade}) • 24 Questions • 90 Mins</p>
                <p className="text-emerald-700 font-semibold">Ready for student taking</p>
              </div>
            </div>
          )}
        </div>

        {/* Footer Navigation Controls */}
        <div className="flex items-center justify-between border-t border-slate-100 bg-slate-50 px-6 py-3">
          <button
            type="button"
            disabled={step === 1}
            onClick={() => setStep((s) => s - 1)}
            className="flex items-center gap-1 text-xs font-semibold text-slate-600 hover:text-slate-900 disabled:opacity-30"
          >
            <ChevronLeft className="h-4 w-4" /> Back
          </button>

          {step < 4 ? (
            <button
              type="button"
              onClick={() => setStep((s) => s + 1)}
              className="flex items-center gap-1 rounded-xl bg-teal-700 px-4 py-2 text-xs font-semibold text-white hover:bg-teal-800"
            >
              Next Step <ChevronRight className="h-4 w-4" />
            </button>
          ) : (
            <button
              type="button"
              onClick={onClose}
              className="flex items-center gap-1 rounded-xl bg-emerald-600 px-4 py-2 text-xs font-semibold text-white hover:bg-emerald-700"
            >
              <CheckCircle2 className="h-4 w-4" /> Publish Exam Now
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
