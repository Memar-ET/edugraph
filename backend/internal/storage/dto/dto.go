package dto

type PresignUploadRequest struct {
	Bucket      string `json:"bucket" validate:"required,oneof=curriculum exam reports audit"`
	Key         string `json:"key" validate:"required"`
	ContentType string `json:"content_type,omitempty"`
}

type PresignDownloadRequest struct {
	Bucket string `json:"bucket" validate:"required,oneof=curriculum exam reports audit"`
	Key    string `json:"key" validate:"required"`
}

type PresignResponse struct {
	URL       string `json:"url"`
	Key       string `json:"key"`
	ExpiresIn int    `json:"expires_in"`
}
