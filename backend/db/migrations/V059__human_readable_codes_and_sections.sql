-- V059: Human-readable institutional identifiers + classroom sections.
--
-- Adds human-readable codes alongside the existing UUID primary keys
-- (never replacing them) for students/teachers/exams -- schools already
-- have `code` (V003). Also adds a lightweight `section` label on
-- students (e.g. 'A'/'B') so a school's same-grade roster can be
-- narratively grouped into classrooms (Grade 9A / Grade 9B) without
-- introducing a new classroom table: nothing in the existing backend or
-- frontend queries "by classroom" today (class-heatmap and every other
-- teacher-facing aggregate is scoped by school+subject+grade, not by a
-- finer classroom unit), so a full parallel classroom/course subsystem
-- would be unused infrastructure this project's own conventions avoid
-- building. `section` is additive/nullable and purely a display/roster
-- grouping.

ALTER TABLE students
    ADD COLUMN IF NOT EXISTS student_code TEXT,
    ADD COLUMN IF NOT EXISTS section      TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_students_student_code ON students (student_code) WHERE student_code IS NOT NULL;

ALTER TABLE teachers
    ADD COLUMN IF NOT EXISTS teacher_code TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_teachers_teacher_code ON teachers (teacher_code) WHERE teacher_code IS NOT NULL;

ALTER TABLE assessment.exams
    ADD COLUMN IF NOT EXISTS exam_code TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_exams_exam_code ON assessment.exams (exam_code) WHERE exam_code IS NOT NULL;
