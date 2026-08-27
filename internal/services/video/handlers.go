package video

import (
	"encoding/json"
	"net/http"
)

// UploadHandler returns an HTTP handler for POST /api/v1/jobs.
func UploadHandler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form with 500MB limit.
		if err := r.ParseMultipartForm(500 * 1024 * 1024); err != nil {
			http.Error(w, "failed to parse upload", http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()

		// Extract video file.
		file, _, err := r.FormFile("video")
		if err != nil {
			http.Error(w, "video file is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Extract form fields.
		sessionID := r.FormValue("session_id")
		if sessionID == "" {
			http.Error(w, "session_id is required", http.StatusBadRequest)
			return
		}

		userID := r.FormValue("user_id")
		if userID == "" {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		deviceID := r.FormValue("device_id")

		// Process upload.
		req := UploadRequest{
			SessionID: sessionID,
			UserID:    userID,
			DeviceID:  deviceID,
			VideoData: file,
		}

		resp, err := svc.Upload(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return 201 Created with job details.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}
