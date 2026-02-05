package httpapi

import (
	"net/http"
	"strconv"
)

func (s *Server) handleOpenAPISubStore(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	pwd := r.URL.Query().Get("pwd")
	tunnel := r.URL.Query().Get("tunnel")
	if tunnel == "" {
		tunnel = "-1"
	}

	if user == "" || pwd == "" {
		writeJSON(w, http.StatusBadRequest, Err("用户或密码不能为空"))
		return
	}

	userInfo, err := s.store.GetUserByName(r.Context(), user)
	if err != nil || userInfo.Pwd != md5Hex(pwd) {
		writeJSON(w, http.StatusUnauthorized, Err("鉴权失败"))
		return
	}

	const giga = 1024 * 1024 * 1024
	var header string
	if tunnel == "-1" {
		header = buildSubscriptionHeader(userInfo.OutFlow, userInfo.InFlow, userInfo.Flow*int64(giga), userInfo.ExpTime/1000)
	} else {
		id, _ := strconv.ParseInt(tunnel, 10, 64)
		ut, err := s.store.GetUserTunnelByID(r.Context(), id)
		if err != nil || ut.UserID != userInfo.ID {
			writeJSON(w, http.StatusBadRequest, Err("隧道不存在"))
			return
		}
		header = buildSubscriptionHeader(ut.OutFlow, ut.InFlow, ut.Flow*int64(giga), ut.ExpTime/1000)
	}

	w.Header().Set("subscription-userinfo", header)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(header))
}

func buildSubscriptionHeader(upload, download, total, expire int64) string {
	return "upload=" + strconv.FormatInt(download, 10) + "; download=" + strconv.FormatInt(upload, 10) + "; total=" + strconv.FormatInt(total, 10) + "; expire=" + strconv.FormatInt(expire, 10)
}
