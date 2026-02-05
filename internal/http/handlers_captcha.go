package httpapi

import (
	"encoding/json"
	"net/http"
)

type captchaVerifyRequest struct {
	ID        string          `json:"id"`
	Data      json.RawMessage `json:"data"`
	CaptchaID string          `json:"captchaId"`
	TrackData json.RawMessage `json:"trackData"`
}

func (s *Server) handleCaptchaCheck(w http.ResponseWriter, r *http.Request) {
	if s.isCaptchaEnabled(r) {
		writeJSON(w, http.StatusOK, OK(1))
		return
	}
	writeJSON(w, http.StatusOK, OK(0))
}

func (s *Server) handleCaptchaGenerate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusBadRequest, Err("仅支持Cloudflare验证码"))
}

func (s *Server) handleCaptchaVerify(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusBadRequest, Err("仅支持Cloudflare验证码"))
}

func writeCaptchaVerify(w http.ResponseWriter, ok bool, token string) {
	resp := map[string]any{
		"code": 4001,
		"msg":  "fail",
	}
	if ok {
		resp["code"] = 200
		resp["msg"] = "success"
		resp["data"] = map[string]any{"validToken": token}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
