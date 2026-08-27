package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/devwillsha/voidcut/internal/auth"
)

// DeviceStartHandler handles POST /api/v1/auth/device/start
func (g *Gateway) DeviceStartHandler(deviceSvc *auth.DeviceLoginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deviceSvc == nil {
			http.Error(w, "device service not configured", http.StatusInternalServerError)
			return
		}

		var req auth.DeviceStartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		resp, err := deviceSvc.Start(r.Context(), req)
		if err != nil {
			if g.log != nil {
				g.log.Errorw("device start failed", "error", err)
			}
			http.Error(w, "failed to start device login", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// DeviceApproveHandler handles POST /api/v1/auth/device/approve
func (g *Gateway) DeviceApproveHandler(deviceSvc *auth.DeviceLoginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deviceSvc == nil {
			http.Error(w, "device service not configured", http.StatusInternalServerError)
			return
		}

		var req auth.DeviceApproveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.DeviceCode == "" || req.UserID == "" {
			http.Error(w, "device_code and user_id are required", http.StatusBadRequest)
			return
		}

		if err := deviceSvc.Approve(r.Context(), req); err != nil {
			if g.log != nil {
				g.log.Errorw("device approve failed", "error", err)
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"approved"}`))
	}
}

// DeviceTokenHandler handles POST /api/v1/auth/device/token
func (g *Gateway) DeviceTokenHandler(deviceSvc *auth.DeviceLoginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deviceSvc == nil {
			http.Error(w, "device service not configured", http.StatusInternalServerError)
			return
		}

		var req auth.DeviceTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.DeviceCode == "" {
			http.Error(w, "device_code is required", http.StatusBadRequest)
			return
		}

		resp, err := deviceSvc.Token(r.Context(), req)
		if err != nil {
			// Check for authorization_pending error.
			if err.Error() == "authorization_pending" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
				return
			}
			if g.log != nil {
				g.log.Errorw("device token failed", "error", err)
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// ConnectHandler handles GET /connect - approval page for device login.
// In Phase 3, this will be a full HTML page with session auth.
// For now, it's a stub that shows the user code.
func (g *Gateway) ConnectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userCode := r.URL.Query().Get("user_code")
		if userCode == "" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body><h1>Device Login</h1><p>Missing user_code</p></body></html>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		html := `<html><body><h1>Approve Device Login</h1>
<p>User Code: <strong>` + userCode + `</strong></p>
<p>In Phase 3, this page will require session login and allow approval.</p>
</body></html>`
		_, _ = w.Write([]byte(html))
	}
}

// MountDeviceAuthRoutes adds device login endpoints to the gateway router.
func (g *Gateway) MountDeviceAuthRoutes(deviceSvc *auth.DeviceLoginService) {
	if g == nil || g.router == nil || deviceSvc == nil {
		return
	}

	g.router.Post("/api/v1/auth/device/start", g.DeviceStartHandler(deviceSvc))
	g.router.Post("/api/v1/auth/device/approve", g.DeviceApproveHandler(deviceSvc))
	g.router.Post("/api/v1/auth/device/token", g.DeviceTokenHandler(deviceSvc))
	g.router.Get("/connect", g.ConnectHandler())
}
