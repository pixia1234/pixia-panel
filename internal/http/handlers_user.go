package httpapi

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"time"

	"pixia-panel/internal/auth"
	"pixia-panel/internal/captcha"
	"pixia-panel/internal/store"
)

type loginRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	CaptchaID      string `json:"captchaId"`
	CaptchaCode    string `json:"captchaCode"`
	TurnstileToken string `json:"turnstileToken"`
}

type userCreateRequest struct {
	User          string `json:"user"`
	Pwd           string `json:"pwd"`
	Flow          int64  `json:"flow"`
	Num           int64  `json:"num"`
	ExpTime       int64  `json:"expTime"`
	FlowResetTime int64  `json:"flowResetTime"`
	Status        *int64 `json:"status"`
}

type userUpdateRequest struct {
	ID            int64  `json:"id"`
	User          string `json:"user"`
	Pwd           string `json:"pwd"`
	Flow          int64  `json:"flow"`
	Num           int64  `json:"num"`
	ExpTime       int64  `json:"expTime"`
	FlowResetTime int64  `json:"flowResetTime"`
	Status        *int64 `json:"status"`
}

type userDeleteRequest struct {
	ID int64 `json:"id"`
}

type changePasswordRequest struct {
	NewUsername     string `json:"newUsername"`
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

type resetFlowRequest struct {
	ID   int64 `json:"id"`
	Type int64 `json:"type"`
}

func (s *Server) handleUserLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, Err("用户名或密码不能为空"))
		return
	}

	if s.isCaptchaEnabled(r) {
		turnstileEnabled, turnstileSecret := s.turnstileConfig(r)
		if turnstileEnabled {
			if req.TurnstileToken == "" {
				writeJSON(w, http.StatusBadRequest, Err("验证码不能为空"))
				return
			}
			resp, err := captcha.VerifyTurnstile(r.Context(), turnstileSecret, req.TurnstileToken, r.RemoteAddr)
			if err != nil || !resp.Success {
				writeJSON(w, http.StatusBadRequest, Err("验证码校验失败"))
				return
			}
		} else {
			if req.CaptchaID == "" {
				writeJSON(w, http.StatusBadRequest, Err("验证码不能为空"))
				return
			}
			if !s.captcha.ConsumeToken(req.CaptchaID) {
				writeJSON(w, http.StatusBadRequest, Err("验证码校验失败"))
				return
			}
		}
	}

	user, err := s.store.GetUserByName(r.Context(), req.Username)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("账号或密码错误"))
		return
	}

	if user.Pwd != md5Hex(req.Password) {
		writeJSON(w, http.StatusBadRequest, Err("账号或密码错误"))
		return
	}
	if user.Status == 0 {
		writeJSON(w, http.StatusBadRequest, Err("账户停用"))
		return
	}

	token, err := auth.Sign(s.jwtSecret, user.ID, user.RoleID, s.tokenTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("登录失败"))
		return
	}

	requirePwdChange := user.User == "admin_user" || req.Password == "admin_user"
	data := map[string]any{
		"token":                 token,
		"name":                  user.User,
		"role_id":               user.RoleID,
		"requirePasswordChange": requirePwdChange,
	}

	writeJSON(w, http.StatusOK, OK(data))
}

func (s *Server) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	var req userCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.User == "" || req.Pwd == "" {
		writeJSON(w, http.StatusBadRequest, Err("用户名或密码不能为空"))
		return
	}
	if _, err := s.store.GetUserByName(r.Context(), req.User); err == nil {
		writeJSON(w, http.StatusBadRequest, Err("用户名已存在"))
		return
	}

	status := int64(1)
	if req.Status != nil {
		status = *req.Status
	}
	user := &store.User{
		User:          req.User,
		Pwd:           md5Hex(req.Pwd),
		RoleID:        1,
		ExpTime:       req.ExpTime,
		Flow:          req.Flow,
		InFlow:        0,
		OutFlow:       0,
		FlowResetTime: req.FlowResetTime,
		Num:           req.Num,
		CreatedTime:   time.Now().UnixMilli(),
		UpdatedTime:   nil,
		Status:        status,
	}

	if _, err := s.store.InsertUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("创建失败"))
		return
	}

	writeJSON(w, http.StatusOK, OK("用户创建成功"))
}

func (s *Server) handleUserList(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("获取失败"))
		return
	}
	for i := range users {
		users[i].Pwd = ""
	}
	writeJSON(w, http.StatusOK, OK(users))
}

func (s *Server) handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	var req userUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	user, err := s.store.GetUserByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("用户不存在"))
		return
	}

	if existing, err := s.store.GetUserByName(r.Context(), req.User); err == nil && existing.ID != req.ID {
		writeJSON(w, http.StatusBadRequest, Err("用户名已被其他用户使用"))
		return
	}

	status := user.Status
	if req.Status != nil {
		status = *req.Status
	}
	var pwd *string
	if req.Pwd != "" {
		h := md5Hex(req.Pwd)
		pwd = &h
	}
	if err := s.store.UpdateUserFields(r.Context(), req.ID, req.User, pwd, req.Flow, req.Num, req.ExpTime, req.FlowResetTime, status, time.Now().UnixMilli()); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("更新失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("用户更新成功"))
}

func (s *Server) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	var req userDeleteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.ID == 0 {
		writeJSON(w, http.StatusBadRequest, Err("ID不能为空"))
		return
	}
	user, err := s.store.GetUserByID(r.Context(), req.ID)
	if err == nil && user.RoleID == 0 {
		writeJSON(w, http.StatusBadRequest, Err("不能删除管理员用户"))
		return
	}
	if err := s.deleteUserCascade(r, req.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, OK("用户及关联数据删除成功"))
}

func (s *Server) handleUserPackage(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromCtx(r)
	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("用户不存在"))
		return
	}
	user.Pwd = ""

	tunnels, _ := s.store.ListUserTunnelsByUser(r.Context(), userID)
	forwards, _ := s.store.ListForwardsByUser(r.Context(), userID)
	stats, _ := s.store.ListRecentStatistics(r.Context(), userID, 24)
	stats = fillLast24(stats, userID)

	data := map[string]any{
		"userInfo":          user,
		"tunnelPermissions": tunnels,
		"forwards":          forwards,
		"statisticsFlows":   stats,
	}
	writeJSON(w, http.StatusOK, OK(data))
}

func (s *Server) handleUserUpdatePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		writeJSON(w, http.StatusBadRequest, Err("新密码和确认密码不匹配"))
		return
	}

	userID := userIDFromCtx(r)
	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("用户不存在"))
		return
	}
	if user.Pwd != md5Hex(req.CurrentPassword) {
		writeJSON(w, http.StatusBadRequest, Err("当前密码错误"))
		return
	}
	if existing, err := s.store.GetUserByName(r.Context(), req.NewUsername); err == nil && existing.ID != userID {
		writeJSON(w, http.StatusBadRequest, Err("用户名已被其他用户使用"))
		return
	}
	newPwd := md5Hex(req.NewPassword)
	if err := s.store.UpdateUserFields(r.Context(), userID, req.NewUsername, &newPwd, user.Flow, user.Num, user.ExpTime, user.FlowResetTime, user.Status, time.Now().UnixMilli()); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("账号密码修改失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("账号密码修改成功"))
}

func (s *Server) handleUserResetFlow(w http.ResponseWriter, r *http.Request) {
	var req resetFlowRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Err("参数错误"))
		return
	}
	if req.Type == 1 {
		user, err := s.store.GetUserByID(r.Context(), req.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Err("用户不存在"))
			return
		}
		user.InFlow = 0
		user.OutFlow = 0
		user.UpdatedTime = ptrInt64(time.Now().UnixMilli())
		if err := s.store.UpdateUser(r.Context(), user); err != nil {
			writeJSON(w, http.StatusInternalServerError, Err("更新失败"))
			return
		}
		writeJSON(w, http.StatusOK, OK("ok"))
		return
	}

	ut, err := s.store.GetUserTunnelByID(r.Context(), req.ID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
		return
	}
	ut.InFlow = 0
	ut.OutFlow = 0
	if err := s.store.UpdateUserTunnel(r.Context(), ut); err != nil {
		writeJSON(w, http.StatusInternalServerError, Err("更新失败"))
		return
	}
	writeJSON(w, http.StatusOK, OK("ok"))
}

func (s *Server) isCaptchaEnabled(r *http.Request) bool {
	cfg, err := s.store.GetConfigByName(r.Context(), "captcha_enabled")
	if err != nil {
		return false
	}
	return cfg.Value == "true"
}

func (s *Server) turnstileConfig(r *http.Request) (bool, string) {
	enabled := false
	if cfg, err := s.store.GetConfigByName(r.Context(), "turnstile_enabled"); err == nil {
		enabled = cfg.Value == "true"
	}
	if !enabled {
		return false, ""
	}
	secret := ""
	if cfg, err := s.store.GetConfigByName(r.Context(), "turnstile_secret_key"); err == nil {
		secret = cfg.Value
	}
	return enabled && secret != "", secret
}

func md5Hex(val string) string {
	h := md5.Sum([]byte(val))
	return hex.EncodeToString(h[:])
}

func ptrInt64(v int64) *int64 {
	return &v
}
