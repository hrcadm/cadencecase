package response

import "github.com/yourname/sleeptracker/internal"

type APIResponse struct {
	Data  interface{}        `json:"data,omitempty"`
	Meta  map[string]any     `json:"meta,omitempty"`
	Error *internal.AppError `json:"error,omitempty"`
}

func Success(data interface{}, meta map[string]any) APIResponse {
	return APIResponse{Data: data, Meta: meta, Error: nil}
}

func BadRequest(msg string) APIResponse {
	return APIResponse{Error: internal.NewAppError(400, msg)}
}

func InternalError(msg string) APIResponse {
	return APIResponse{Error: internal.NewAppError(500, msg)}
}

func NotFound(msg string) APIResponse {
	return APIResponse{Error: internal.NewAppError(404, msg)}
}

func NewAppError(status int, msg string) APIResponse {
	return APIResponse{Error: internal.NewAppError(status, msg)}
}
