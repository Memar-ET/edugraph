package dto

// ResyncCLOsResult is returned by POST /curriculum/clos/resync.
type ResyncCLOsResult struct {
	Synced int `json:"synced"`
	Failed int `json:"failed"`
}
