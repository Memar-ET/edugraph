import {
  BarChart3,
  Briefcase,
  ClipboardList,
  FileStack,
  GitBranch,
  HelpCircle,
  LayoutDashboard,
  MessageCircleQuestion,
  Network,
  Settings,
  ShieldAlert,
  UploadCloud,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import type { Role } from '@/types/api'

export interface NavItem {
  label: string
  to: string
  icon: LucideIcon
  badge?: string | number
  section?: 'General' | 'Reports' | 'Settings'
}

export function getNavItems(role: Role | undefined): NavItem[] {
  switch (role) {
    case 'student':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard, section: 'General' },
        { label: 'My Exams', to: '/student/exams', icon: ClipboardList, section: 'General', badge: '6' },
        { label: 'Ask AI Tutor', to: '/student/tutor', icon: MessageCircleQuestion, section: 'General' },
        { label: 'Career Paths', to: '/career/paths', icon: Briefcase, section: 'General' },
        { label: 'Analytics', to: '/', icon: BarChart3, section: 'Reports' },
        { label: 'Help & Support', to: '/', icon: HelpCircle, section: 'Settings' },
        { label: 'Settings', to: '/', icon: Settings, section: 'Settings' },
      ]
    case 'teacher':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard, section: 'General' },
        { label: 'Exam Suite', to: '/teacher/exams/upload', icon: UploadCloud, section: 'General' },
        { label: 'Class Analytics', to: '/', icon: BarChart3, section: 'Reports' },
        { label: 'Help & Support', to: '/', icon: HelpCircle, section: 'Settings' },
        { label: 'Settings', to: '/', icon: Settings, section: 'Settings' },
      ]
    case 'school_admin':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard, section: 'General' },
        { label: 'Exam Suite', to: '/teacher/exams/upload', icon: UploadCloud, section: 'General' },
        { label: 'Quality Score', to: '/', icon: ShieldAlert, section: 'Reports' },
        { label: 'Help & Support', to: '/', icon: HelpCircle, section: 'Settings' },
        { label: 'Settings', to: '/', icon: Settings, section: 'Settings' },
      ]
    case 'regional_admin':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard, section: 'General' },
        { label: 'Regional Analytics', to: '/', icon: BarChart3, section: 'Reports' },
        { label: 'Help & Support', to: '/', icon: HelpCircle, section: 'Settings' },
        { label: 'Settings', to: '/', icon: Settings, section: 'Settings' },
      ]
    case 'ministry_admin':
      return [
        { label: 'Dashboard', to: '/', icon: LayoutDashboard, section: 'General' },
        { label: 'Curriculum Specification', to: '/curriculum/upload', icon: FileStack, section: 'General' },
        { label: 'Prerequisites', to: '/curriculum/prerequisites', icon: Network, section: 'General' },
        { label: 'Curriculum Versions', to: '/curriculum/versions', icon: GitBranch, section: 'General' },
        { label: 'Career Paths', to: '/career/paths', icon: Briefcase, section: 'General' },
        { label: 'National Reports', to: '/', icon: BarChart3, section: 'Reports' },
        { label: 'Help & Support', to: '/', icon: HelpCircle, section: 'Settings' },
        { label: 'Settings', to: '/', icon: Settings, section: 'Settings' },
      ]
    case 'curriculum_officer':
      return [
        { label: 'Dashboard', to: '/curriculum', icon: LayoutDashboard, section: 'General' },
        { label: 'Upload Curriculum', to: '/curriculum/upload', icon: FileStack, section: 'General' },
        { label: 'Prerequisites', to: '/curriculum/prerequisites', icon: Network, section: 'General' },
        { label: 'Curriculum Versions', to: '/curriculum/versions', icon: GitBranch, section: 'General' },
        { label: 'Parsing Analytics', to: '/', icon: BarChart3, section: 'Reports' },
        { label: 'Help & Support', to: '/', icon: HelpCircle, section: 'Settings' },
        { label: 'Settings', to: '/', icon: Settings, section: 'Settings' },
      ]
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
