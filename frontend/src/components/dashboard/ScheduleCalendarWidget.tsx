import { useState } from 'react'
import { ChevronLeft, ChevronRight, Plus, Calendar, Clock, ChevronRight as ArrowRight, X } from 'lucide-react'
import { cn } from '@lib/utils/cn'
import { ThreeDCard } from '@components/ui'

export interface ScheduleItem {
  id: string
  title: string
  time: string
  subtitle: string
  category: 'schedule' | 'upcoming'
  typeBadge?: string
}

export interface ScheduleCalendarWidgetProps {
  title?: string
  monthLabel?: string
  days?: Array<{ dayName: string; dateNumber: number; isActive?: boolean }>
  scheduleItems: ScheduleItem[]
  onAddNew?: () => void
  className?: string
}

const DEFAULT_DAYS = [
  { dayName: 'Mon', dateNumber: 11 },
  { dayName: 'Tue', dateNumber: 12 },
  { dayName: 'Wed', dateNumber: 13 },
  { dayName: 'Thu', dateNumber: 14 },
  { dayName: 'Fri', dateNumber: 15 },
]

export function ScheduleCalendarWidget({
  title = 'Schedule & Events',
  monthLabel = 'April 2026',
  days = DEFAULT_DAYS,
  scheduleItems,
  onAddNew,
  className,
}: ScheduleCalendarWidgetProps) {
  const [selectedDate, setSelectedDate] = useState<number>(11)
  const [itemsList, setItemsList] = useState<ScheduleItem[]>(scheduleItems)
  const [showAddModal, setShowAddModal] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newTime, setNewTime] = useState('')
  const [newSubtitle, setNewSubtitle] = useState('')

  const activeItems = itemsList.filter((i) => i.category === 'schedule')
  const upcomingItems = itemsList.filter((i) => i.category === 'upcoming')

  const handleAddEvent = (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTitle.trim()) return

    const newItem: ScheduleItem = {
      id: `custom-${Date.now()}`,
      title: newTitle.trim(),
      time: newTime.trim() || '10:00 AM - 11:00 AM',
      subtitle: newSubtitle.trim() || 'AI Scheduled Task',
      category: 'schedule',
    }

    setItemsList([newItem, ...itemsList])
    setNewTitle('')
    setNewTime('')
    setNewSubtitle('')
    setShowAddModal(false)
    if (onAddNew) onAddNew()
  }

  return (
    <>
      <ThreeDCard
        className={cn('flex flex-col border-gray-100/90 bg-white p-5', className)}
      >
        {/* Month Header & Add Button */}
        <div className="flex items-center justify-between pb-4" title={title}>
          <div className="flex items-center gap-1.5 font-display text-base font-bold text-gray-900">
            <span>{monthLabel}</span>
            <ChevronRight className="h-4 w-4 text-gray-400" />
          </div>
          <button
            type="button"
            onClick={() => setShowAddModal(true)}
            className="inline-flex items-center gap-1.5 rounded-xl bg-gray-900 px-3 py-1.5 text-xs font-bold text-white shadow-sm transition-all hover:bg-gray-800 hover:scale-105 active:scale-95"
          >
            <Plus className="h-3.5 w-3.5" />
            <span>Add New</span>
          </button>
        </div>

        {/* Horizontal Date Picker Strip */}
        <div className="flex items-center justify-between gap-1 rounded-2xl bg-gray-50 p-1.5 border border-gray-100">
          <button
            type="button"
            onClick={() => setSelectedDate((d) => Math.max(10, d - 1))}
            className="flex h-8 w-6 items-center justify-center text-gray-400 hover:text-gray-900"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>

          <div className="flex flex-1 justify-between gap-1">
            {days.map((d) => {
              const isSelected = selectedDate === d.dateNumber
              return (
                <button
                  key={d.dateNumber}
                  type="button"
                  onClick={() => setSelectedDate(d.dateNumber)}
                  className={cn(
                    'flex flex-col items-center justify-center rounded-xl py-1.5 px-2.5 transition-all text-xs font-medium cursor-pointer',
                    isSelected
                      ? 'bg-gray-900 text-white shadow-md font-bold scale-105'
                      : 'text-gray-600 hover:bg-gray-200/80',
                  )}
                >
                  <span className="text-[10px] opacity-80">{d.dayName}</span>
                  <span className="text-sm font-extrabold mt-0.5">{d.dateNumber}</span>
                </button>
              )
            })}
          </div>

          <button
            type="button"
            onClick={() => setSelectedDate((d) => Math.min(16, d + 1))}
            className="flex h-8 w-6 items-center justify-center text-gray-400 hover:text-gray-900"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>

        {/* Schedule Items List */}
        <div className="mt-4 space-y-4">
          {/* Schedule Section */}
          <div>
            <div className="flex items-center justify-between text-xs font-bold text-gray-900 mb-2">
              <span>Schedule for Apr {selectedDate}</span>
              <button type="button" className="text-gray-400 hover:text-gray-900 font-semibold">
                See All
              </button>
            </div>

            <div className="space-y-2">
              {activeItems.map((item) => (
                <div
                  key={item.id}
                  className="group flex items-center justify-between rounded-xl border border-gray-100 bg-gray-50/70 p-3 transition-all hover:bg-gray-100/90 hover:border-gray-300 hover:scale-[1.01] cursor-pointer"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gray-200 text-gray-800 group-hover:bg-gray-900 group-hover:text-white transition-all shadow-xs">
                      <Calendar className="h-4 w-4" />
                    </div>
                    <div>
                      <h4 className="text-xs font-bold text-gray-900">{item.title}</h4>
                      <div className="flex items-center gap-2 text-[11px] text-gray-500 mt-0.5">
                        <Clock className="h-3 w-3" />
                        <span>{item.time}</span>
                      </div>
                      <p className="text-[11px] text-gray-400 font-medium">{item.subtitle}</p>
                    </div>
                  </div>
                  <ArrowRight className="h-4 w-4 text-gray-400 group-hover:text-gray-900 group-hover:translate-x-1 transition-all" />
                </div>
              ))}
            </div>
          </div>

          {/* Upcoming Section */}
          <div>
            <div className="flex items-center justify-between text-xs font-bold text-gray-900 mb-2">
              <span>Upcoming Tasks</span>
              <button type="button" className="text-gray-400 hover:text-gray-900 font-semibold">
                See All
              </button>
            </div>

            <div className="space-y-2">
              {upcomingItems.map((item) => (
                <div
                  key={item.id}
                  className="group flex items-center justify-between rounded-xl border border-gray-100 bg-white p-3 transition-all hover:bg-gray-50 hover:border-gray-300 hover:scale-[1.01] cursor-pointer shadow-2xs"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gray-100 text-gray-600 group-hover:bg-gray-900 group-hover:text-white transition-all">
                      <Clock className="h-4 w-4" />
                    </div>
                    <div>
                      <h4 className="text-xs font-bold text-gray-900">{item.title}</h4>
                      <p className="text-[11px] text-gray-500 font-medium">{item.time}</p>
                      <p className="text-[11px] text-gray-400 font-medium">{item.subtitle}</p>
                    </div>
                  </div>
                  <ArrowRight className="h-4 w-4 text-gray-400 group-hover:text-gray-900 group-hover:translate-x-1 transition-all" />
                </div>
              ))}
            </div>
          </div>
        </div>
      </ThreeDCard>

      {/* Interactive Event Addition Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-gray-900/40 backdrop-blur-sm p-4 animate-in fade-in">
          <div className="w-full max-w-md rounded-2xl border border-gray-100 bg-white p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between border-b border-gray-100 pb-3">
              <h3 className="font-display text-lg font-bold text-gray-900">Add New Event / Task</h3>
              <button
                type="button"
                onClick={() => setShowAddModal(false)}
                className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-900"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <form onSubmit={handleAddEvent} className="space-y-3">
              <div>
                <label className="block text-xs font-bold text-gray-700 mb-1">Event Title</label>
                <input
                  type="text"
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  placeholder="Physics Midterm / AI Tutor Review..."
                  className="w-full rounded-xl border border-gray-200 bg-gray-50 px-3.5 py-2 text-xs text-gray-900 font-medium outline-none focus:border-gray-900 focus:bg-white"
                  required
                />
              </div>
              <div>
                <label className="block text-xs font-bold text-gray-700 mb-1">Time & Duration</label>
                <input
                  type="text"
                  value={newTime}
                  onChange={(e) => setNewTime(e.target.value)}
                  placeholder="10:00 AM - 11:30 AM"
                  className="w-full rounded-xl border border-gray-200 bg-gray-50 px-3.5 py-2 text-xs text-gray-900 font-medium outline-none focus:border-gray-900 focus:bg-white"
                />
              </div>
              <div>
                <label className="block text-xs font-bold text-gray-700 mb-1">Instructor / Topic Subtitle</label>
                <input
                  type="text"
                  value={newSubtitle}
                  onChange={(e) => setNewSubtitle(e.target.value)}
                  placeholder="Devon Lane / Unit 3 Kinematics"
                  className="w-full rounded-xl border border-gray-200 bg-gray-50 px-3.5 py-2 text-xs text-gray-900 font-medium outline-none focus:border-gray-900 focus:bg-white"
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="rounded-xl border border-gray-200 bg-white px-4 py-2 text-xs font-bold text-gray-700 hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="rounded-xl bg-gray-900 px-4 py-2 text-xs font-bold text-white hover:bg-gray-800"
                >
                  Save Schedule Event
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  )
}
