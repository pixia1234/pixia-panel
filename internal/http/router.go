package httpapi

import "net/http"

func NewRouter(flowHandler *FlowHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/flow/upload", flowHandler.Upload)
	mux.HandleFunc("/flow/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test"))
	})
	return mux
}
