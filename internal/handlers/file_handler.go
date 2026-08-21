package handlers

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

type FileHandler struct {
	Store store.FileStore
}

func NewFileHandler(s store.FileStore) *FileHandler {
	return &FileHandler{Store: s}
}

// maxUploadSize caps a single file at 20 MB — generous for photos and
// PDFs, and enough for a short video clip, while keeping database rows
// reasonable since files are stored directly in Postgres (no external
// object storage is configured for this deployment).
const maxUploadSize = 20 << 20 // 20 MB

var allowedContentTypePrefixes = []string{"image/", "video/", "application/pdf"}

func isAllowedContentType(ct string) bool {
	for _, prefix := range allowedContentTypePrefixes {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}

// UploadFile handles POST /api/files/upload (multipart/form-data, field "file")
// Accepts images, videos, and PDFs — used by every module that needs an
// attachment (ticket photos, engineer resumes/documents, timesheet
// proof, client requirement files, social media assets).
func (h *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+1<<20) // small buffer over the limit for multipart overhead

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or invalid upload (max 20MB): "+err.Error())
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field: "+err.Error())
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if !isAllowedContentType(contentType) {
		writeError(w, http.StatusBadRequest, "unsupported file type: "+contentType+" — only images, videos, and PDFs are accepted")
		return
	}
	if header.Size > maxUploadSize {
		writeError(w, http.StatusBadRequest, "file too large — max 20MB")
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed reading upload: "+err.Error())
		return
	}

	claims := ClaimsFromContext(r.Context())
	uploadedBy := ""
	if claims != nil {
		uploadedBy = claims.UserID
	}

	f := &models.File{
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        int64(len(data)),
		UploadedBy:  uploadedBy,
		Data:        data,
	}
	if err := h.Store.Save(f); err != nil {
		writeError(w, http.StatusInternalServerError, "failed saving file: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":           f.ID,
		"filename":     f.Filename,
		"content_type": f.ContentType,
		"size":         f.Size,
		"url":          "/api/files/" + f.ID,
	})
}

// DownloadFile handles GET /api/files/{id}
// Streams the file back with its original content type, so images and
// PDFs render inline in the browser and videos can be played directly.
func (h *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	f, err := h.Store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))
	w.Header().Set("Content-Disposition", `inline; filename="`+f.Filename+`"`)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(f.Data)
}
