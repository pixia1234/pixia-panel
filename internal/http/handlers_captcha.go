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
	challenge := s.captcha.Generate()
	resp := map[string]any{
		"code": 200,
		"msg":  "success",
		"id":   challenge.ID,
		"captcha": map[string]any{
			"type":                  challenge.Type,
			"backgroundImage":       challenge.BackgroundImage,
			"templateImage":         challenge.TemplateImage,
			"backgroundImageHeight": challenge.BackgroundImageHeight,
			"data":                  challenge.Data,
		},
		"data": map[string]any{
			"id": challenge.ID,
			"captcha": map[string]any{
				"type":                  challenge.Type,
				"backgroundImage":       challenge.BackgroundImage,
				"templateImage":         challenge.TemplateImage,
				"backgroundImageHeight": challenge.BackgroundImageHeight,
				"data":                  challenge.Data,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCaptchaVerify(w http.ResponseWriter, r *http.Request) {
	var req captchaVerifyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeCaptchaVerify(w, false, "")
		return
	}

	id := req.ID
	if id == "" {
		id = req.CaptchaID
	}
	trackData := req.Data
	if len(trackData) == 0 {
		trackData = req.TrackData
	}
	if len(trackData) > 0 && trackData[0] == '"' {
		var s string
		if err := json.Unmarshal(trackData, &s); err == nil {
			trackData = []byte(s)
		}
	}

	if id == "" || len(trackData) == 0 {
		writeCaptchaVerify(w, false, "")
		return
	}
	if s.captcha.VerifyTrack(id, trackData) {
		writeCaptchaVerify(w, true, id)
		return
	}
	writeCaptchaVerify(w, false, "")
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
