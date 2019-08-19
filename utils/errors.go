package utils

import (
	"net/http"
)

type StatusError interface {
	StatusCode() int
	Status() string
	Error() string
}

type ForesightError struct {
	code    int
	status  string
	message string
}

func NewForesightError(code int, status, message string) StatusError {
	return &ForesightError{
		code:    code,
		status:  status,
		message: message,
	}
}

func (e *ForesightError) StatusCode() int {
	return e.code
}

func (e *ForesightError) Status() string {
	return e.status
}

func (e *ForesightError) Error() string {
	return e.message
}

var (
	AuthWithoutCookie     = NewForesightError(http.StatusUnauthorized, "COOKIE_MISSING", "access denied since no cookie")
	AuthWithInvalidCookie = NewForesightError(http.StatusUnauthorized, "INVALID_COOKIE", "invalid cookie")
	ParamsMismatch        = NewForesightError(http.StatusBadRequest, "BAD_REQUEST", "params mismatch")
	InvalidFile           = NewForesightError(http.StatusBadRequest, "BAD_REQUEST", "invalid file")
	TargetObjectNotFound  = NewForesightError(http.StatusNotFound, "NOT_FOUND", "target not found")
	DatabaseQueryError    = NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query db")
	DatabaseUpdateError   = NewForesightError(http.StatusInternalServerError, "DB_UPDATE_ERROR", "error on update database")
	DatabaseInsertError   = NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	DatabaseDeleteError   = NewForesightError(http.StatusInternalServerError, "DB_DELETE_ERROR", "error on delete data")
	FileOpError           = NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on operating file")
	SystemOpError         = NewForesightError(http.StatusInternalServerError, "SYSTEM_OP_ERROR", "error on call system operation")
	NetworkError          = NewForesightError(http.StatusInternalServerError, "NETWORK_ERROR", "error on transform data on internet")
)
