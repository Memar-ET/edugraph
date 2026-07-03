package dto

type OverviewResponse struct {
	TotalRegions  int64 `json:"total_regions"`
	TotalSchools  int64 `json:"total_schools"`
	TotalStudents int64 `json:"total_students"`
	TotalTeachers int64 `json:"total_teachers"`
}

type RegionStatsResponse struct {
	RegionID           string  `json:"region_id"`
	SchoolCount        int64   `json:"school_count"`
	StudentCount       int64   `json:"student_count"`
	TeacherCount       int64   `json:"teacher_count"`
	AvgAssessmentScore float64 `json:"avg_assessment_score"`
}
