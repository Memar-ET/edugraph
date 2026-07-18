package service

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	gradeLevelRe = regexp.MustCompile(`(?i)grade\s*(\d{1,2})`)
	// \btest\b (not a plain Contains) so "Test 1"/"Test" match but "Fastest"
	// or "Latest" don't.
	bareTestRe = regexp.MustCompile(`(?i)\btest\b`)
	// Matches "Units 1-2", "Unit 3", "Units 1–3" (en dash) etc.
	unitRangeRe = regexp.MustCompile(`(?i)units?\s+(\d{1,2})\s*(?:[-\x{2013}]\s*(\d{1,2}))?`)
)

// allowedTotalMarks enforces the mark totals from Capability 2A's spec:
// Unit Test 10 or 15, Midterm 15 or 20, Final Exam 40.
var allowedTotalMarks = map[string][]int{
	"unit_test":  {10, 15},
	"midterm":    {15, 20},
	"final_exam": {40},
}

// deriveGradeAndScope extracts the grade level and exam scope from a
// teacher-written exam title, e.g. "Grade 11 Biology Unit Test - Cell
// Biology" -> (11, "unit_test", true). There's no manual override field for
// either -- the title is the single source of truth for what the exam
// covers, matched against curriculum.subjects by
// repository.MatchSubjectFromTitle.
func deriveGradeAndScope(title string) (gradeLevel int, examScope string, ok bool) {
	lower := strings.ToLower(title)

	m := gradeLevelRe.FindStringSubmatch(lower)
	if m == nil {
		return 0, "", false
	}
	gradeLevel, err := strconv.Atoi(m[1])
	if err != nil || gradeLevel < 1 || gradeLevel > 12 {
		return 0, "", false
	}

	examScope = deriveExamScope(lower)
	if examScope == "" {
		return 0, "", false
	}
	return gradeLevel, examScope, true
}

func deriveExamScope(lowerTitle string) string {
	switch {
	case strings.Contains(lowerTitle, "final"):
		return "final_exam"
	case strings.Contains(lowerTitle, "midterm"), strings.Contains(lowerTitle, "mid-term"), strings.Contains(lowerTitle, "mid term"):
		return "midterm"
	case strings.Contains(lowerTitle, "unit test"), strings.Contains(lowerTitle, "unit exam"), strings.Contains(lowerTitle, "quiz"), bareTestRe.MatchString(lowerTitle):
		return "unit_test"
	default:
		return ""
	}
}

// deriveUnitNumbers extracts which curriculum.units a Unit Test/Midterm
// covers from the title, e.g. "Units 1-3" -> [1,2,3], "Unit 4" -> [4]. A
// final_exam title like "All Units" won't match at all -- that's the
// correct outcome (nil means "whole subject/grade" for final_exam, or
// "scope not resolved, skip the scope-narrowed check" for unit_test/
// midterm; see service/exam_validate.go). This is a fallback guess at
// upload time -- ai-service/app/services/exam_parser/metadata.py applies
// the same pattern to the document's own metadata table when present,
// which is authoritative and corrects this (same two-stage pattern already
// used for subject/grade/scope/marks).
func deriveUnitNumbers(title string) []int {
	m := unitRangeRe.FindStringSubmatch(title)
	if m == nil {
		return nil
	}
	start, err := strconv.Atoi(m[1])
	if err != nil {
		return nil
	}
	end := start
	if m[2] != "" {
		end, err = strconv.Atoi(m[2])
		if err != nil || end < start {
			return []int{start}
		}
	}
	units := make([]int, 0, end-start+1)
	for n := start; n <= end; n++ {
		units = append(units, n)
	}
	return units
}

// totalMarksAllowed checks a declared total against the scope's allowed
// set. Cross-checking this later against the sum of parsed question marks
// is 2B's job (AI Validation), not this capability's.
func totalMarksAllowed(examScope string, totalMarks int) bool {
	for _, m := range allowedTotalMarks[examScope] {
		if m == totalMarks {
			return true
		}
	}
	return false
}
