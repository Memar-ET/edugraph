# EduGraph AI — Frontend Architecture & UI/UX Specification (`ui.md`)

> **Version**: 2.0.0  
> **Target Framework**: React 18 + TypeScript + Vite + Tailwind CSS + TanStack Router & Query + Zustand  
> **Design Philosophy**: High-End Slate/Neutral Monochrome Aesthetic with Data-Dense Glassmorphic Cards & Role-Tailored Interfaces  

---

## 1. Executive Summary & Design System Architecture

### 1.1 Overview
**EduGraph AI** is a national-scale curriculum-intelligence and assessment platform designed for Ethiopian K-12 education. The frontend provides a unified, role-based single-page application (SPA) serving six distinct user personas:
1. **Students**: Personalized mastery dashboards, AI-generated study plans, Graph-RAG AI tutor, and interactive examination workspace.
2. **Teachers**: Exam creation & parsing suite, psychometric exam quality analytics, bulk grading interface, and class-wide prerequisite gap heatmaps.
3. **School Administrators**: School quality scorecards, teacher/student rosters, and institutional performance monitoring.
4. **Regional Administrators**: Zonal school comparisons, regional quality indices, and resource allocation insights.
5. **Ministry Admins**: National education overview, curriculum version control oversight, cross-regional analytics, and prerequisite resync controls.
6. **Curriculum Officers**: Multi-stage curriculum document upload, AI-extracted AST tree review/editing, mid-year subject versioning, and Neo4j graph promotion.

---

### 1.2 Design Aesthetics & Visual Tokens

The user interface follows a modern grey-scale / slate neutral design palette with subtle, deliberate color highlights for semantic status. Inspired by state-of-the-art analytics dashboards, it features rounded cards, crisp typography, clean data visualisations, search bars, filters, and role-based action widgets.

#### Color Palette System
```scss
/* Core Neutral Palette (Slate/Zinc scale) */
--color-bg-app:          #f8fafc;  /* slate-50 background */
--color-surface-card:    #ffffff;  /* pure white card surface */
--color-surface-hover:   #f1f5f9;  /* slate-100 highlight */
--color-border-subtle:   #e2e8f0;  /* slate-200 border */
--color-border-strong:   #cbd5e1;  /* slate-300 border */
--color-text-primary:    #0f172a;  /* slate-900 heading/primary text */
--color-text-secondary:  #475569;  /* slate-600 body text */
--color-text-muted:      #94a3b8;  /* slate-400 subtle label text */

/* Brand & Role Accent Colors */
--color-brand-primary:   #1e293b;  /* slate-800 deep slate accent */
--color-brand-accent:    #0f766e;  /* teal-700 primary brand highlight */
--color-brand-light:     #f0fdf4;  /* emerald-50 surface highlight */

/* Semantic Status Tokens */
--color-status-success:  #059669;  /* emerald-600 for high mastery/approved */
--color-status-warning:  #d97706;  /* amber-600 for moderate gap/pending */
--color-status-danger:   #dc2626;  /* red-600 for critical struggle/flagged */
--color-status-info:     #2563eb;  /* blue-600 for AI processing/info */

/* Glassmorphic & Shadow Tokens */
--shadow-card:           0 1px 3px 0 rgba(0, 0, 0, 0.05), 0 1px 2px -1px rgba(0, 0, 0, 0.05);
--shadow-card-hover:     0 10px 15px -3px rgba(0, 0, 0, 0.08), 0 4px 6px -4px rgba(0, 0, 0, 0.03);
--backdrop-blur-glass:   blur(12px);
```

#### Typography Scale (Google Fonts: Inter & Outfit)
- **Display Font (`font-display`)**: *Outfit*, sans-serif (Headers, Stat numbers, Hero text)
- **Body Font (`font-sans`)**: *Inter*, sans-serif (Interface text, inputs, code snippets)
- **Hierarchy**:
  - `Display XL`: 32px / Bold / Leading 1.2 (`text-3xl font-bold font-display`)
  - `Display LG`: 24px / Semibold / Leading 1.3 (`text-2xl font-semibold font-display`)
  - `Title`: 18px / Semibold / Leading 1.4 (`text-lg font-semibold`)
  - `Body`: 14px / Regular / Leading 1.5 (`text-sm font-normal`)
  - `Caption`: 12px / Medium / Leading 1.4 (`text-xs font-medium text-slate-500`)

---

## 2. Technology Stack & Directory Architecture

### 2.1 Core Stack Dependencies
| Package | Version | Purpose |
|---|---|---|
| `react` | `^18.3.1` | Core UI library |
| `vite` | `^5.4.0` | Next-gen build tool & dev server |
| `@tanstack/react-router` | `^1.50.0` | Type-safe client-side routing & navigation guards |
| `@tanstack/react-query` | `^5.51.0` | Server state management, caching & background polling |
| `zustand` | `^4.5.4` | Client-side persistent state (Auth session, theme, UI state) |
| `tailwindcss` | `^3.4.9` | Utility-first styling engine with custom theme tokens |
| `lucide-react` | `^0.424.0` | Clean vector iconography |
| `recharts` | `^2.12.7` | Data visualisations (Heatmaps, gauge meters, bar charts) |
| `clsx` / `tailwind-merge` | `^2.4.0` | Conditional class composition utility (`cn`) |

---

### 2.2 Frontend Directory Structure
```
frontend/src/
├── app/
│   ├── providers/
│   │   ├── QueryProvider.tsx          # TanStack Query Client configuration
│   │   └── ToastProvider.tsx          # System notifications toast host
│   └── router.tsx                     # TanStack Router route tree & role guards
├── components/
│   ├── charts/
│   │   ├── DistributionBars.tsx       # Horizontal/vertical grade breakdown chart
│   │   ├── HeatmapGrid.tsx            # Topic struggle matrix (Neo4j graph powered)
│   │   └── ScoreGauge.tsx             # Circular SVG score meter component
│   ├── forms/
│   │   ├── FormInput.tsx              # Standard input with inline validation error
│   │   ├── FormSelect.tsx             # Styled select dropdown
│   │   └── FileDropzone.tsx           # Drag & drop upload target with magic-byte check
│   ├── layout/
│   │   ├── AppHeader.tsx              # Sticky header with breadcrumbs & search
│   │   ├── AppShell.tsx               # Main layout wrapper with responsive sidebar
│   │   ├── NavConfig.ts               # Role-based menu links mapping
│   │   └── NotificationBell.tsx       # Live alert bell dropdown popover
│   ├── shared/
│   │   ├── StatusBadge.tsx            # Color-coded badge (Approved, Parsing, Pending)
│   │   ├── LoadingSkeleton.tsx        # Skeleton placeholders for async cards/tables
│   │   └── ConfirmModal.tsx           # Reusable modal dialog for destructive actions
│   └── ui/
│       ├── Button.tsx                 # Variants: primary, secondary, ghost, danger
│       ├── Card.tsx                   # Styled surface container with subtle border
│       ├── Input.tsx                  # Core text input
│       └── Tabs.tsx                   # Accessible tab switcher
├── features/
│   ├── assessment/                    # Exam upload, review, taking, grading & quality
│   ├── auth/                          # Login, password reset, demo user switcher
│   ├── career/                        # Career path discovery & AI match generator
│   ├── curriculum/                    # File upload, AI tree review, version control
│   ├── ministry/                      # National overview & regional breakdown
│   ├── regional/                      # Zonal monitoring dashboard
│   ├── school-admin/                  # Institutional quality score & roster management
│   ├── student/                       # Subject health, study plans & Graph-RAG AI tutor
│   └── teacher/                       # Teacher dashboard & class gap heatmap analytics
├── i18n/
│   ├── am.json                        # Amharic translation strings
│   ├── en.json                        # English translation strings
│   └── useTranslation.ts              # Translation hook
├── lib/
│   ├── api/
│   │   ├── client.ts                  # Axios instance with JWT refresh interceptor
│   │   └── endpoints.ts               # Type-safe API client functions
│   └── utils/
│       └── cn.ts                      # Classmerge utility
├── stores/
│   └── auth.store.ts                  # Auth state, token storage, user roles & claims
├── types/
│   ├── api.types.ts                   # DTO definitions matching Go backend
│   └── domain.types.ts                # Frontend domain primitives
├── index.css                          # Tailwind directives & CSS custom properties
└── main.tsx                           # Application entry point
```

---

## 3. Navigation Framework & Shared Shell Layout

### 3.1 AppShell Layout Architecture

The `AppShell` component (`frontend/src/components/layout/AppShell.tsx`) acts as the persistent layout frame for all authenticated routes.

```
+-----------------------------------------------------------------------------------+
|  [EduGraph Logo]       | [Search anything... 🔍]               [🔔 (3)] [Profile ▼] |
+------------------------+----------------------------------------------------------+
|                        |                                                          |
|  GENERAL               |  Page Header Title                                       |
|  - Dashboard           |  Description subtitle text                               |
|  - Curriculum          |  ------------------------------------------------------  |
|  - Exams               |                                                          |
|  - AI Tutor            |  +-------------------+  +-------------------+  +--------+ |
|  - Career Paths        |  | Stat Card 1       |  | Stat Card 2       |  | Stat 3 | |
|                        |  +-------------------+  +-------------------+  +--------+ |
|  REPORTS & INSIGHTS    |                                                          |
|  - Class Heatmap       |  +-----------------------------------------------------+ |
|  - Quality Score       |  | Main Page Content / Data Table / Chart Grid            | |
|  - Regional Stats      |  |                                                     | |
|                        |  +-----------------------------------------------------+ |
|  SETTINGS              |                                                          |
|  - Help & Support      |                                                          |
|  [Sign Out]            |                                                          |
+------------------------+----------------------------------------------------------+
```

#### Key Shell Components:
1. **Sidebar Ledger**:
   - Fixed width (`256px` / `w-64`), crisp white background with `border-r border-slate-200`.
   - Role-filtered navigation links loaded dynamically from `getNavItems(user.role)`.
   - Active route highlighting: `border-l-4 border-slate-800 bg-slate-100 text-slate-900 font-semibold`.
   - User identity profile tile at footer displaying user full name, role badge, and Sign Out button.

2. **Top Sticky Bar**:
   - Height `64px` (`h-16`), backdrop blur effect (`bg-white/90 backdrop-blur-md border-b border-slate-200`).
   - Mobile navigation toggle burger button (`lg:hidden`).
   - Dynamic page context title & subtitle description.
   - Global contextual search input box with keyboard shortcut hint (`Ctrl + K`).
   - Live `NotificationBell` displaying unread badge count with popover list for system events (e.g. school flagged for review, parsing completed).

---

## 4. Comprehensive Page & Workflow Specifications

---

### 4.1 Auth Feature (`/login`)

#### Page Purpose
Secure login interface supporting institutional accounts, JWT token exchange, refresh token rotation, and single-click demo persona switcher for testing.

#### Layout Wireframe
```
+-----------------------------------------------------------------------------------+
|                                                                                   |
|     [ EduGraph AI ]                                                               |
|     Curriculum & Assessment Intelligence Platform                                 |
|                                                                                   |
|     +-----------------------------------------------------------------------+     |
|     |  Sign In to Your Account                                              |     |
|     |                                                                       |     |
|     |  Email Address                                                        |     |
|     |  [ email@edugraph.et                                                ] |     |
|     |                                                                       |     |
|     |  Password                                                             |     |
|     |  [ **********                                                       ] |     |
|     |                                                                       |     |
|     |  [  Sign In  ]                                                        |     |
|     |                                                                       |     |
|     |  ------------------ Quick Demo Persona Selector ------------------    |     |
|     |  [Student] [Teacher] [School Admin] [Regional] [Ministry] [Officer]   |     |
|     +-----------------------------------------------------------------------+     |
|                                                                                   |
+-----------------------------------------------------------------------------------+
```

#### API Endpoints & State
- `POST /api/v1/auth/login` → returns `{ access_token, refresh_token, user: { id, full_name, email, role } }`
- Stores token in `useAuthStore` with local storage persistence.
- Automatic routing based on `landingPathFor(user.role)`:
  - `student` → `/` (Student Dashboard)
  - `teacher` → `/` (Teacher Dashboard)
  - `school_admin` → `/` (School Admin Dashboard)
  - `regional_admin` → `/` (Regional Dashboard)
  - `ministry_admin` → `/` (Ministry Dashboard)
  - `curriculum_officer` → `/curriculum` (Curriculum Dashboard)

---

### 4.2 Curriculum Domain (Curriculum Officer & Ministry)

#### 1. Curriculum Officer Dashboard (`/curriculum`)
- **API Call**: `GET /api/v1/curriculum/jobs?page=1&limit=10`
- **Key Features**:
  - **Summary Stat Metrics**: Total Jobs Uploaded, Pending Review Jobs, Promoted Subject Count, AI Extraction Accuracy Rate.
  - **Upload Job History Table**: Columns for Job ID, File Name, Subject/Grade/Year, Status (`pending`, `parsing`, `parsed`, `review`, `approved`, `rejected`, `failed`), Submitted Date, Actions (Review Tree button).
  - **Mid-Year Revision Lineage Panel**: Displays active subject versions (`is_current = true`), links to version history modal (`GET /curriculum/subjects/{code}/versions`), and provides "Supersede Subject" modal launcher.

#### 2. Curriculum Upload Workspace (`/curriculum/upload`)
- **API Call**: `POST /api/v1/curriculum/upload` (Form Data: `file`, `subject_code`, `grade`, `academic_year`)
- **Key Features**:
  - **Drag-and-Drop Dropzone**: Supports `.pdf` and `.docx` files up to 50MB.
  - **Server-Side Magic Byte Validation**: Sniffs magic bytes client/server side to prevent mime spoofing.
  - **Live Processing Status**: Upon upload, shows job enqueue notification (`queue:curriculum:parse`) with automatic redirect option to `JobReviewPage`.

#### 3. Curriculum AST Tree Review & Graph Promotion Page (`/curriculum/jobs/$jobId`)
- **API Calls**:
  - `GET /api/v1/curriculum/jobs/{id}` (Returns parsed AST JSON structure)
  - `GET /api/v1/storage/files/{jobId}` (Streams raw document blob for integrated PDF preview)
  - `POST /api/v1/curriculum/jobs/{id}/approve` (Promotes AST to Postgres `curriculum` schema & Neo4j graph)
  - `POST /api/v1/curriculum/topics/{id}/prerequisites` (Adds topic prerequisite dependency edge)
  - `PATCH /api/v1/curriculum/topics/{id}/prerequisites/{prereqId}/validate` (Confirms AI-inferred link)

#### Split-Screen Layout Wireframe (`JobReviewPage`)
```
+-----------------------------------------------------------------------------------+
|  [< Back to Dashboard]  Job #8492 - Physics Grade 11    [ Reject ]  [ Approve & Promote ]|
+--------------------------------------------------+--------------------------------+
| INTEGRATED DOCUMENT PREVIEW                      | CURRICULUM TREE EDITOR         |
| +----------------------------------------------+ | [Expand All] [Collapse All]    |
| | Page 1 / 42                 [Zoom +] [Zoom-] | |                                |
| |                                              | | ▼ Unit 1: Kinematics           |
| |  MINISTRY OF EDUCATION                       | |   ├── Topic 1.1: Motion in 1D |
| |  CURRICULUM SPECIFICATION                    | |   │   ├── CLO 11.1.1 (Vector) |
| |                                              | |   │   └── [Edit] [Delete]      |
| |  Unit 1: Kinematics                          | |   ├── Topic 1.2: Vectors     |
| |  1.1 Motion in One Dimension                 | |   │   └── Prereq: Topic 9.4 🔗 |
| |  Students will understand speed...           | |   └── [+ Add Topic]            |
| |                                              | |                                |
| +----------------------------------------------+ | ▼ Unit 2: Dynamics             |
|                                                  |   └── Topic 2.1: Newton Laws   |
+--------------------------------------------------+--------------------------------+
```

---

### 4.3 Assessment Domain (Teacher Workspace)

#### 1. Teacher Dashboard (`/` for Role `teacher`)
- **API Call**: `GET /api/v1/teachers/me/class-heatmap`
- **Key Features**:
  - **Metric Stat Cards**: Active Classes Count, Total Exams Created, Class Average Mastery %, High-Struggle Topic Alerts.
  - **Class-Wide Gap Heatmap Grid (`HeatmapGrid`)**:
    - X-Axis: Topics from Neo4j prerequisite graph.
    - Y-Axis: Enrolled Students in school/class.
    - Cell Colors: Emerald (`mastered`), Amber (`moderate gap`), Ruby Red (`severe gap / root cause`).
  - **Cross-Grade Alert Banner**: Triggers automated alert when >40% of students struggle with a foundational prerequisite topic (walked via `HAS_PREREQUISITE*1..3` graph).

#### 2. Exam Upload & Parsing Page (`/teacher/exams/upload`)
- **API Calls**:
  - `POST /api/v1/exams/upload` (Pushes exam to `queue:exam:parse`)
  - `POST /api/v1/exams/{id}/answer-key` (Pushes answer key to `queue:exam:answerkey`)
- **Key Features**:
  - Dual file picker for Exam paper PDF/DOCX and official Answer Key document.
  - Grade level & subject target selectors.
  - Real-time status tracker for AI question alignment and CLO auto-mapping.

#### 3. Exam Review & CLO Alignment Page (`/teacher/exams/$examId`)
- **API Calls**: `GET /api/v1/exams/{id}`, `POST /api/v1/exams/{id}/publish`
- **Key Features**:
  - Editable table of parsed questions (Question Text, Options A-D, Correct Answer, Aligned CLO Code, Stated Difficulty Level).
  - One-click "Publish Exam" toggle to make the exam active for student taking.

#### 4. Bulk Exam Grading Matrix (`/teacher/exams/$examId/grade`)
- **API Calls**:
  - `GET /api/v1/exams/{id}/grading-questions`
  - `POST /api/v1/exams/{id}/grades/bulk` (Submits question scores + `timeSpentSecs`)
- **Key Features**:
  - Interactive grid for manual rubric scoring of short-answer and essay questions.
  - Automatic calculation of CLO mastery score updates upon submission.

#### 5. Psychometric Exam Quality Analytics (`/teacher/exams/$examId/quality`)
- **API Call**: `GET /api/v1/exams/{id}/quality`
- **Key Features**:
  - **Discrimination Index Bar**: Measures item discrimination power (higher score distinguishes high vs. low performers).
  - **Difficulty Calibration Chart**: Stated difficulty vs. actual empirical student performance calibration.
  - **Timing Anomaly Detector**: Highlights questions where average `timeSpentSecs` deviates significantly from expected pacing.
  - **Mandatory CLO Coverage Gauge (`ScoreGauge`)**: Visual percentage gauge of curriculum standards assessed.

---

### 4.4 Student Domain (Personalized Learning & AI Tutor)

#### 1. Student Command Center (`/` for Role `student`)
- **API Calls**:
  - `GET /api/v1/students/me/subject-profiles`
  - `GET /api/v1/students/me/study-plans`
- **Layout Wireframe**:
```
+-----------------------------------------------------------------------------------+
|  Welcome back, Abebe 👋                                                           |
|  Grade 11 Student - Addis Ababa Secondary                                         |
+-----------------------------------------------------------------------------------+
|  STAT METRICS                                                                     |
|  +--------------------+  +--------------------+  +--------------------+           |
|  | Overall Mastery    |  | Active Study Plan  |  | Identified Gaps    |           |
|  |  78%  [+3.2% ▲]    |  |  Day 3 of 7        |  |  4 Topics          |           |
|  +--------------------+  +--------------------+  +--------------------+           |
|                                                                                   |
|  SUBJECT HEALTH LAYER (Bilingual EN/AM)                                           |
|  +------------------------------------------------------------------------------+  |
|  | Physics          [ Mastery: 82% ]  ████████████████████░░░░░  (Healthy)      |  |
|  | Chemistry        [ Mastery: 64% ]  ██████████████░░░░░░░░░░░  (Gap Detected) |  |
|  | Mathematics      [ Mastery: 91% ]  █████████████████████████  (Strong)       |  |
|  +------------------------------------------------------------------------------+  |
|                                                                                   |
|  ACTIVE STUDY PLAN (Kahn Topological Sort Ordered)                                |
|  +------------------------------------------------------------------------------+  |
|  | [Day 1: Vectors] -> [Day 2: Kinematics] -> [Day 3 (Current): Newton's 2nd Law] |  |
|  | Root Cause Prioritization: Vector Math required for Newton's 2nd Law          |  |
|  +------------------------------------------------------------------------------+  |
+-----------------------------------------------------------------------------------+
```

#### 2. Student Exam Workspace (`/student/exams` & `/student/exams/$examId`)
- **API Calls**:
  - `GET /api/v1/exams/{id}/questions`
  - `POST /api/v1/exams/{id}/submit`
- **Key Features**:
  - Distraction-free exam taking mode with fixed header timer.
  - Per-question answer recorder tracking individual answer selections and duration (`timeSpentSecs`).
  - Automatic submission on timer expiration with instant trigger of `queue:gap:analyze`.

#### 3. Graph-RAG AI Tutor Chat Interface (`/student/tutor`)
- **API Call**: `POST /api/v1/tutor/ask` (Payload: `{ question, topic_id }`)
- **Key Features**:
  - **Dual-Panel Interface**: Left panel: Chat history with markdown formatting, LaTeX formula rendering ($E = mc^2$), and code highlighting. Right panel: Graph-RAG Context Drawer revealing the exact prerequisite chain and student gap history passed into Gemini.
  - **Bilingual Amharic / English Toggle**: Single-click toggle for Amharic or English responses.

---

### 4.5 Institutional Oversight (School, Regional & Ministry)

#### 1. School Admin Dashboard (`/` for Role `school_admin`)
- **API Call**: `GET /api/v1/schools/{id}/quality-scores`
- **Key Features**:
  - **School Composite Quality Score Card**: Breakdown across CLO Coverage, Student Mastery %, Exam Psychometric Quality, and Ministry Compliance Score (Redis cached with 1h TTL).
  - Teacher roster & student enrollment management tables (`/teachers`, `/students`).

#### 2. Regional Admin Dashboard (`/` for Role `regional_admin`)
- **API Call**: `GET /api/v1/ministry/regions/{regionID}/stats`
- **Key Features**:
  - Comparative zonal performance map & table.
  - School quality score rankings across regional districts.

#### 3. Ministry Command Dashboard (`/` for Role `ministry_admin`)
- **API Calls**:
  - `GET /api/v1/ministry/overview`
  - `POST /api/v1/curriculum/prerequisites/resync`
- **Key Features**:
  - National distribution charts (`DistributionBars`) for student mastery across all 11 Ethiopian regions.
  - **Prerequisite Graph Bulk Resync Button**: One-click bulk sync triggering `POST /curriculum/prerequisites/resync` to sync un-mirrored prerequisite edges into Neo4j graph DB.

---

### 4.6 Career Guidance Domain (`/career/paths`)

#### Page Purpose
AI-powered career matching engine correlating student CLO mastery, subject strengths, and interest vectors with national career pathways.

#### Key Features & API Integration
- **API Calls**:
  - `GET /api/v1/career/paths` (Lists available national career profiles)
  - `POST /api/v1/students/{studentID}/career/generate` (Triggers AI matcher)
  - `GET /api/v1/students/{studentID}/career/matches` (Retrieves generated matches)
- **Visual Matcher UI**:
  - Career Match Cards showing title (e.g. *Software Engineer*, *Civil Engineer*, *Agronomist*), required CLO prerequisites, alignment score gauge (`ScoreGauge`), and recommended remedial subjects.

---

## 5. State Management & API Integration Architecture

### 5.1 Auth Store (`useAuthStore`)
State stored via Zustand with `localStorage` persistence:
```typescript
interface AuthState {
  user: UserClaims | null
  accessToken: string | null
  refreshToken: string | null
  setAuth: (user: UserClaims, accessToken: string, refreshToken: string) => void
  clearAuth: () => void
}
```

### 5.2 Axios API Interceptor (`lib/api/client.ts`)
- Automatically attaches `Authorization: Bearer <accessToken>` header to all outgoing requests under `/api/v1`.
- Handles `401 Unauthorized` status by issuing a background request to `POST /api/v1/auth/refresh` using the stored refresh token.
- If refresh succeeds, retries original failed request seamlessly; if refresh fails, executes `clearAuth()` and redirects to `/login`.

---

## 6. Verification & Quality Assurance Plan

### 6.1 Automated Frontend Build & Type Validation
Before considering any frontend feature complete, run the following verification pipeline:
```bash
# 1. TypeScript Strict Typecheck
npm run type-check

# 2. ESLint Static Code Analysis
npm run lint

# 3. Production Bundle Build Verification
npm run build
```

### 6.2 Manual UI Verification Checklist
- [ ] **Role Routing Guard**: Verify each role lands on its dedicated landing path and is blocked from restricted routes.
- [ ] **Responsive Breakpoints**: Test layout rendering at `375px` (Mobile), `768px` (Tablet), and `1440px` (Desktop).
- [ ] **Theme & Aesthetics**: Confirm visual alignment with slate/neutral grey aesthetic, rounded cards, and proper contrast ratios.
- [ ] **Graph & Chart Rendering**: Verify `HeatmapGrid`, `ScoreGauge`, and `DistributionBars` render correctly with empty, partial, and full demo data.
