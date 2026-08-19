import {
  Award,
  BarChart3,
  Bell,
  BookOpen,
  Briefcase,
  Building2,
  CheckSquare,
  ClipboardList,
  Download,
  FileBarChart2,
  FileStack,
  FlaskConical,
  GitBranch,
  GraduationCap,
  LayoutDashboard,
  LayoutGrid,
  Lightbulb,
  ListTree,
  Map,
  MessageCircleQuestion,
  Monitor,
  Network,
  ScrollText,
  Settings,
  ShieldAlert,
  Target,
  UploadCloud,
  Users,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import type { Role } from '@/types/api'

// Navigation is grouped by user intent, not object type.
// Applied loosely per role — only sections that have items are rendered.
export const NAV_SECTIONS = ['Act', 'Manage', 'Understand', 'Govern'] as const
export type NavSection = (typeof NAV_SECTIONS)[number]

export interface NavItem {
  label: string
  to: string
  icon: LucideIcon
  badge?: string | number
  section: NavSection
}

export function getNavItems(role: Role | undefined): NavItem[] {
  switch (role) {
    // ── Student ───────────────────────────────────────────────────────
    case 'student':
      return [
        { label: 'Home',                 to: '/',                             icon: LayoutDashboard,     section: 'Act' },
        { label: 'Study Plan',           to: '/student/study-plans',          icon: BookOpen,            section: 'Act' },
        { label: 'Exam Center',          to: '/student/exams',                icon: ClipboardList,       section: 'Act' },
        { label: 'Subject Profiles',     to: '/student/subject-profiles',     icon: BarChart3,           section: 'Manage' },
        { label: 'Career Explorer',      to: '/career/paths',                 icon: Briefcase,           section: 'Manage' },
        { label: 'Skill Map',            to: '/student/skill-map',            icon: Network,             section: 'Understand' },
        { label: 'Misconception Journal',to: '/student/misconception-journal',icon: Lightbulb,           section: 'Understand' },
        { label: 'Achievements',         to: '/student/achievements',         icon: Award,               section: 'Understand' },
        { label: 'AI Tutor',             to: '/student/tutor',                icon: MessageCircleQuestion,section: 'Understand' },
        { label: 'Settings',             to: '/settings',                     icon: Settings,            section: 'Govern' },
      ]

    // ── Teacher ───────────────────────────────────────────────────────
    case 'teacher':
      return [
        { label: 'Dashboard',            to: '/',                             icon: LayoutDashboard,     section: 'Act' },
        { label: 'Exams',                to: '/teacher/exams',                icon: ClipboardList,       section: 'Act' },
        { label: 'Students',             to: '/teacher/students',             icon: Users,               section: 'Manage' },
        { label: 'Question Bank',        to: '/teacher/question-bank',        icon: ListTree,            section: 'Manage' },
        { label: 'Announcements',        to: '/teacher/announcements',        icon: Bell,                section: 'Manage' },
        { label: 'Class Analytics',      to: '/teacher/class-analytics',      icon: BarChart3,           section: 'Understand' },
        { label: 'Misconception Review', to: '/teacher/misconceptions',       icon: Lightbulb,           section: 'Understand' },
        { label: 'Curriculum Coverage',  to: '/teacher/curriculum-coverage',  icon: BookOpen,            section: 'Understand' },
        { label: 'Reports',              to: '/teacher/reports',              icon: FileBarChart2,       section: 'Understand' },
        { label: 'Settings',             to: '/settings',                     icon: Settings,            section: 'Govern' },
      ]

    // ── School Admin ──────────────────────────────────────────────────
    case 'school_admin':
      return [
        { label: 'Dashboard',            to: '/',                             icon: LayoutDashboard,     section: 'Act' },
        { label: 'Teacher Roster',       to: '/school/teachers',              icon: GraduationCap,       section: 'Manage' },
        { label: 'Student Roster',       to: '/school/students',              icon: Users,               section: 'Manage' },
        { label: 'Classes & Sections',   to: '/school/classes',               icon: LayoutGrid,          section: 'Manage' },
        { label: 'Announcements',        to: '/school/announcements',         icon: Bell,                section: 'Manage' },
        { label: 'Quality Score',        to: '/school/quality',               icon: ShieldAlert,         section: 'Understand' },
        { label: 'School Analytics',     to: '/school/analytics',             icon: BarChart3,           section: 'Understand' },
        { label: 'Reports',              to: '/school/reports',               icon: FileBarChart2,       section: 'Understand' },
        { label: 'Approvals',            to: '/school/approvals',             icon: CheckSquare,         section: 'Govern' },
        { label: 'Settings',             to: '/settings',                     icon: Settings,            section: 'Govern' },
      ]

    // ── Regional Admin ────────────────────────────────────────────────
    case 'regional_admin':
      return [
        { label: 'Dashboard',            to: '/',                             icon: LayoutDashboard,     section: 'Act' },
        { label: 'Schools Directory',    to: '/regional/schools',             icon: Building2,           section: 'Manage' },
        { label: 'Announcements',        to: '/regional/announcements',       icon: Bell,                section: 'Manage' },
        { label: 'Regional Analytics',   to: '/regional/analytics',           icon: BarChart3,           section: 'Understand' },
        { label: 'Quality Oversight',    to: '/regional/quality-oversight',   icon: ShieldAlert,         section: 'Understand' },
        { label: 'Resource & Devices',   to: '/regional/resources',           icon: Monitor,             section: 'Understand' },
        { label: 'Reports',              to: '/regional/reports',             icon: FileBarChart2,       section: 'Understand' },
        { label: 'Settings',             to: '/settings',                     icon: Settings,            section: 'Govern' },
      ]

    // ── Ministry Admin ────────────────────────────────────────────────
    case 'ministry_admin':
      return [
        { label: 'Dashboard',            to: '/',                             icon: LayoutDashboard,     section: 'Act' },
        { label: 'Regional Overview',    to: '/ministry/regional-overview',   icon: Map,                 section: 'Manage' },
        { label: 'Announcements',        to: '/ministry/announcements',       icon: Bell,                section: 'Manage' },
        { label: 'National Reports',     to: '/ministry/reports',             icon: BarChart3,           section: 'Understand' },
        { label: 'Quality Governance',   to: '/ministry/quality-governance',  icon: ShieldAlert,         section: 'Understand' },
        { label: 'Curriculum Governance',to: '/ministry/curriculum',          icon: FileStack,           section: 'Govern' },
        { label: 'Data Exports',         to: '/ministry/data-exports',        icon: Download,            section: 'Govern' },
        { label: 'Audit Log',            to: '/ministry/audit-log',           icon: ScrollText,          section: 'Govern' },
        { label: 'Settings',             to: '/settings',                     icon: Settings,            section: 'Govern' },
      ]

    // ── Curriculum Officer ────────────────────────────────────────────
    case 'curriculum_officer':
      return [
        { label: 'Dashboard',            to: '/curriculum',                   icon: LayoutDashboard,     section: 'Act' },
        { label: 'Parsing & Upload',     to: '/curriculum/upload',            icon: UploadCloud,         section: 'Act' },
        { label: 'Curriculum Graph',     to: '/curriculum/graph-explorer',    icon: Network,             section: 'Manage' },
        { label: 'CLO Management',       to: '/curriculum/clo-management',    icon: Target,              section: 'Manage' },
        { label: 'Prerequisite Editor',  to: '/curriculum/prerequisites',     icon: GitBranch,           section: 'Manage' },
        { label: 'Q-Matrix Quality',     to: '/curriculum/qmatrix-quality',   icon: FlaskConical,        section: 'Understand' },
        { label: 'Content Review',       to: '/curriculum/content-review',    icon: ListTree,            section: 'Govern' },
        { label: 'Publish & Versions',   to: '/curriculum/versions',          icon: FileStack,           section: 'Govern' },
        { label: 'Settings',             to: '/settings',                     icon: Settings,            section: 'Govern' },
      ]

    default:
      return []
  }
}

export const ROLE_LABELS: Record<Role, string> = {
  student:           'Student',
  teacher:           'Teacher',
  school_admin:      'School Admin',
  regional_admin:    'Regional Admin',
  ministry_admin:    'Ministry Admin',
  curriculum_officer:'Curriculum Officer',
}
