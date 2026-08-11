// Package fileformat provides server-side file-type detection by magic
// bytes, shared by any handler that accepts curriculum/exam document
// uploads (PDF or DOCX) -- never trust the client-supplied Content-Type
// header, it's attacker-controlled and trivially spoofed. See architecture
// doc §9.6: "MIME type validated server-side (magic bytes), not by
// Content-Type header."
package fileformat

import (
	"bytes"
	"io"
	"mime/multipart"
)

var (
	pdfMagic     = []byte("%PDF-")
	zipMagicA    = []byte{0x50, 0x4B, 0x03, 0x04} // PK\x03\x04 -- standard zip local file header
	zipMagicB    = []byte{0x50, 0x4B, 0x05, 0x06} // PK\x05\x06 -- empty zip archive
	zipMagicSpan = []byte{0x50, 0x4B, 0x07, 0x08} // PK\x07\x08 -- spanned zip archive
)

// SniffPDFOrDOCX peeks at the first few bytes of the uploaded file to
// determine its real type by magic number, then rewinds the reader so the
// full content is still available for storage.Upload. DOCX files are
// OOXML/zip containers, so any of the standard zip magic sequences are
// accepted as a DOCX candidate -- the AI-service side parser will reject
// anything that isn't actually a valid Word package.
func SniffPDFOrDOCX(file multipart.File) (mimeType string, ok bool) {
	head := make([]byte, 8)
	n, _ := io.ReadFull(file, head)
	head = head[:n]

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", false
	}

	switch {
	case bytes.HasPrefix(head, pdfMagic):
		return "application/pdf", true
	case bytes.HasPrefix(head, zipMagicA), bytes.HasPrefix(head, zipMagicB), bytes.HasPrefix(head, zipMagicSpan):
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document", true
	default:
		return "", false
	}
}
