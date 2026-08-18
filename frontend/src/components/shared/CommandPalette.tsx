import { useState, useEffect } from 'react'
import {
  Search,
  BookOpen,
  ClipboardList,
  Building2,
  Brain,
  FileText,
  Command,
  X,
  ChevronRight,
  Shield,
} from 'lucide-react'
import { useNavigate } from '@tanstack/react-router'
import { useAuthStore } from '@stores/auth.store'

interface CommandItem {
  id: string
  title: string
  subtitle: string
  category: 'Actions' | 'Navigation' | 'Search Results'
  icon: typeof Search
  to: string
  roles?: string[]
}

const COMMAND_ITEMS: CommandItem[] = [
  { id: 'cmd-1', title: 'Open Graph-RAG AI Tutor', subtitle: 'Ask questions about prerequisite concept chains', category: 'Actions', icon: Brain, to: '/student/tutor', roles: ['student'] },
  { id: 'cmd-2', title: 'Create New Assessment Exam', subtitle: 'Launch exam authoring wizard', category: 'Actions', icon: ClipboardList, to: '/teacher/exams', roles: ['teacher'] },
  { id: 'cmd-3', title: 'Inspect Curriculum AST Tree', subtitle: 'Open Neo4j topic DAG visualizer', category: 'Navigation', icon: BookOpen, to: '/curriculum/graph-explorer', roles: ['curriculum_officer', 'ministry_admin'] },
  { id: 'cmd-4', title: 'School Quality Scorecard', subtitle: 'Institutional accreditation metrics', category: 'Navigation', icon: Shield, to: '/school/quality', roles: ['school_admin'] },
  { id: 'cmd-5', title: 'Zonal Resource & School Box Monitor', subtitle: 'Check edge node sync latencies', category: 'Navigation', icon: Building2, to: '/regional/resources', roles: ['regional_admin'] },
  { id: 'cmd-6', title: 'National Educational Data Exporter', subtitle: 'Bulk dataset export in CSV/JSON', category: 'Navigation', icon: FileText, to: '/ministry/data-exports', roles: ['ministry_admin'] },
]

export function CommandPalette({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const [query, setQuery] = useState('')

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        if (isOpen) onClose()
        else setQuery('')
      }
      if (e.key === 'Escape' && isOpen) {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen) return null

  const filteredItems = COMMAND_ITEMS.filter((item) => {
    const matchesRole = !item.roles || (user?.role && item.roles.includes(user.role))
    const matchesQuery = item.title.toLowerCase().includes(query.toLowerCase()) || item.subtitle.toLowerCase().includes(query.toLowerCase())
    return matchesRole && matchesQuery
  })

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 bg-slate-900/60 backdrop-blur-sm p-4">
      <div className="w-full max-w-xl rounded-2xl border border-slate-200 bg-white shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <div className="flex items-center gap-3 border-b border-slate-100 px-4 py-3">
          <Search className="h-4 w-4 text-slate-400 shrink-0" />
          <input
            type="text"
            autoFocus
            placeholder="Type a command or search (e.g. 'exam', 'tutor', 'quality')..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full text-sm text-slate-900 placeholder-slate-400 focus:outline-none"
          />
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="max-h-80 overflow-y-auto p-2">
          {filteredItems.length > 0 ? (
            <div className="space-y-1">
              {filteredItems.map((item) => {
                const Icon = item.icon
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => {
                      onClose()
                      void navigate({ to: item.to as string })
                    }}
                    className="w-full flex items-center justify-between rounded-xl px-3 py-2.5 text-left hover:bg-slate-50 transition-colors group"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-teal-50 text-teal-700">
                        <Icon className="h-4 w-4" />
                      </div>
                      <div className="min-w-0">
                        <p className="font-semibold text-xs text-slate-900 truncate">{item.title}</p>
                        <p className="text-[11px] text-slate-500 truncate">{item.subtitle}</p>
                      </div>
                    </div>
                    <ChevronRight className="h-4 w-4 text-slate-300 group-hover:text-teal-600 transition-colors" />
                  </button>
                )
              })}
            </div>
          ) : (
            <div className="py-8 text-center text-xs text-slate-400">
              No matching commands or pages found.
            </div>
          )}
        </div>

        <div className="flex items-center justify-between border-t border-slate-100 bg-slate-50 px-4 py-2 text-[10px] text-slate-400">
          <div className="flex items-center gap-2">
            <span className="flex items-center gap-1 font-mono rounded bg-white px-1.5 py-0.5 border border-slate-200">
              <Command className="h-3 w-3" /> K
            </span>
            <span>to toggle palette</span>
          </div>
          <span>Esc to close</span>
        </div>
      </div>
    </div>
  )
}
