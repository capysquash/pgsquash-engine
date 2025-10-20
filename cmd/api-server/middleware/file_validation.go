package middleware

import (
	"net/http"
	"strings"
)

const (
	MaxFileSize = 10 * 1024 * 1024 // 10MB
	MaxFiles    = 100
)

// FileUploadValidationMiddleware validates file uploads
func FileUploadValidationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" && r.Method != "PUT" {
				next.ServeHTTP(w, r)
				return
			}

			contentType := r.Header.Get("Content-Type")
			if !strings.Contains(contentType, "multipart/form-data") {
				next.ServeHTTP(w, r)
				return
			}

			// Parse multipart form with size limit
			if err := r.ParseMultipartForm(MaxFileSize); err != nil {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"File too large or invalid","code":"FILE_SIZE_LIMIT"}`, http.StatusRequestEntityTooLarge)
				return
			}

			if r.MultipartForm == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Validate file count
			totalFiles := 0
			for _, files := range r.MultipartForm.File {
				totalFiles += len(files)
			}
			if totalFiles > MaxFiles {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Too many files","code":"TOO_MANY_FILES"}`, http.StatusBadRequest)
				return
			}

			// Validate file types
			for _, files := range r.MultipartForm.File {
				for _, file := range files {
					if !strings.HasSuffix(strings.ToLower(file.Filename), ".sql") {
						w.Header().Set("Content-Type", "application/json")
						http.Error(w, `{"error":"Only .sql files allowed","code":"INVALID_FILE_TYPE"}`, http.StatusBadRequest)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
