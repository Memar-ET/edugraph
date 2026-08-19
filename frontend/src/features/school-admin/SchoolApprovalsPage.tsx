import { ClipboardList } from 'lucide-react'

import { AppShell } from '@components/layout'
import { Banner, EmptyState } from '@components/ui'

// Curriculum job approval (parsed/review -> approved) is exclusively a
// curriculum_officer/ministry_admin workflow (GET /curriculum/jobs is
// role-gated to those two roles, and scoped server-side to the caller's
// own uploads besides -- see backend/internal/curriculum/repository/
// job_list.go). Curriculum isn't a school-scoped concept at all
// (upload_jobs has no school_id), so there's no legitimate "school-level
// curriculum approval" data to show here. This page previously called
// that endpoint anyway, which 403'd for every school_admin. There is
// also no backend capability today for the thing "school approvals"
// more plausibly means -- reviewing new teacher/student account
// requests at this school -- so rather than silently 403 or show
// curriculum-officer data, this is an honest "not built yet" state.
export function SchoolApprovalsPage() {
  return (
    <AppShell
      title="Approvals"
      description="Review pending account and enrollment requests for your school."
    >
      <div className="space-y-5">
        <Banner tone="info">
          Curriculum content approval is handled by curriculum officers and isn't scoped to individual schools.
        </Banner>
        <EmptyState
          icon={ClipboardList}
          title="No approval queue yet"
          description="Account and enrollment approval requests for your school aren't available yet."
        />
      </div>
    </AppShell>
  )
}
