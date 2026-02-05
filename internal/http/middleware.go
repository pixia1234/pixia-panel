package httpapi

import (
	"context"
	"net/http"
	"strings"
	"time"

	"pixia-panel/internal/auth"
)

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxRoleID
)

func withAuth(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, Err("未登录"))
			return
		}

		tokenStr := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
		claims, err := auth.Parse(secret, tokenStr)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, Err("登录无效"))
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxRoleID, claims.RoleID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roleID, ok := r.Context().Value(ctxRoleID).(int64)
		if !ok || roleID != 0 {
			writeJSON(w, http.StatusForbidden, Err("权限不足"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func userIDFromCtx(r *http.Request) int64 {
	if v, ok := r.Context().Value(ctxUserID).(int64); ok {
		return v
	}
	return 0
}

func roleIDFromCtx(r *http.Request) int64 {
	if v, ok := r.Context().Value(ctxRoleID).(int64); ok {
		return v
	}
	return 0
}

func withTimeout(next http.Handler, d time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
