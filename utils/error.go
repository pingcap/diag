package utils

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
