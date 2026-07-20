import {
  Briefcase,
  ClipboardList,
  FileStack,
  LayoutDashboard,
  MessageCircleQuestion,
  UploadCloud,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import type { Role } from '@/types/api'

export interface NavItem {
  label: string
  to: string
  icon: LucideIcon
}

/** Sidebar nav is role-scoped -- mirrors router.go's RequireRole gates, not
 * a generic "everything" menu. Each role only sees what it can act on. */
export function getNavItems(role: Role | undefined): NavItem[] {
  switch (role) {
    case 'student':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard },
        { label: 'My exams', to: '/student/exams', icon: ClipboardList },
        { label: 'Ask the tutor', to: '/student/tutor', icon: MessageCircleQuestion },
      ]
    case 'teacher':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard },
        { label: 'Exams', to: '/teacher/exams/upload', icon: UploadCloud },
      ]
    case 'school_admin':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard },
        { label: 'Exams', to: '/teacher/exams/upload', icon: UploadCloud },
      ]
    case 'regional_admin':
      return [{ label: 'Dashboard', to: '/', icon: LayoutDashboard }]
    case 'ministry_admin':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard },
        { label: 'Curriculum', to: '/curriculum/upload', icon: FileStack },
        { label: 'Career paths', to: '/career/paths', icon: Briefcase },
      ]
    case 'curriculum_officer':
      return [{ label: 'Upload curriculum', to: '/curriculum/upload', icon: FileStack }]
    default:
      return []
  }
}

export const ROLE_LABELS: Record<Role, string> = {
  student: 'Student',
  teacher: 'Teacher',
  school_admin: 'School Admin',
  regional_admin: 'Regional Admin',
  ministry_admin: 'Ministry Admin',
  curriculum_officer: 'Curriculum Officer',
}
