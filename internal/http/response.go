package httpapi

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func OK(data interface{}) Response {
	return Response{Code: 0, Msg: "success", Data: data}
}

func Err(msg string) Response {
	return Response{Code: 1, Msg: msg}
}
