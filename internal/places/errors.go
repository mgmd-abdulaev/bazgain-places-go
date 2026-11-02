package places

import "net/http"

type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  []FieldError
	Cause   error
}

type FieldError struct{ Field, Code, Message string }

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Cause }

func NotFound() *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "place_not_found", Message: "Place not found"}
}
func CannotPublishWithoutGallery() *AppError {
	return &AppError{Status: http.StatusConflict, Code: "cannot_publish_place_without_gallery", Message: "A place must have at least one gallery item before it can be published"}
}
func CannotPublishWithoutNavigationPoint() *AppError {
	return &AppError{Status: http.StatusConflict, Code: "cannot_publish_place_without_navigation_point", Message: "A place must have a navigation point before it can be published"}
}
func Validation(field, code, message string) *AppError {
	return &AppError{Status: http.StatusUnprocessableEntity, Code: "validation_error", Message: "Invalid UUID", Fields: []FieldError{{Field: field, Code: code, Message: message}}}
}
func Unauthorized() *AppError {
	return &AppError{Status: http.StatusUnauthorized, Code: "access_token_invalid", Message: "Access token is invalid or expired"}
}
func Forbidden() *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "forbidden", Message: "Forbidden"}
}
func RateLimited() *AppError {
	return &AppError{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: "Too many requests"}
}
func RouteNotFound() *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "not_found", Message: "Route not found"}
}
func MethodNotAllowed() *AppError {
	return &AppError{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: "Method not allowed"}
}
func Internal(cause error) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: "internal_error", Message: "Internal server error", Cause: cause}
}
