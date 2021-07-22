package err

import (
	"go-drive/common/i18n"
	"go-drive/common/types"
	"net/http"
)

type RequestError interface {
	Code() int
	Error() string
}

type RequestErrorWithData interface {
	RequestError
	Data() types.M
}

// BadRequestError 400
type BadRequestError struct {
	msg string
}

func (b BadRequestError) Error() string {
	return b.msg
}

func (b BadRequestError) Code() int {
	return http.StatusBadRequest
}

// UnauthorizedError 401
type UnauthorizedError struct {
	msg string
}

func (i UnauthorizedError) Error() string {
	return i.msg
}

func (i UnauthorizedError) Code() int {
	return http.StatusUnauthorized
}

// NotFoundError 404
type NotFoundError struct {
	msg string
}

func (d NotFoundError) Error() string {
	return d.msg
}

func (d NotFoundError) Code() int {
	return http.StatusNotFound
}

// NotAllowedError 403
type NotAllowedError struct {
	msg string
}

func (d NotAllowedError) Error() string {
	return d.msg
}

func (d NotAllowedError) Code() int {
	return http.StatusForbidden
}

// PermissionDeniedError 403
type PermissionDeniedError struct {
	msg string
}

func (p PermissionDeniedError) Code() int {
	return http.StatusForbidden
}

func (p PermissionDeniedError) Error() string {
	return p.msg
}

// UnsupportedError 405
type UnsupportedError struct {
	msg string
}

func (n UnsupportedError) Error() string {
	return n.msg
}

func (n UnsupportedError) Code() int {
	return http.StatusMethodNotAllowed
}

type RemoteApiError struct {
	code int
	msg  string
}

func (r RemoteApiError) Error() string {
	return r.msg
}

func (r RemoteApiError) Code() int {
	return r.code
}

func IsUnauthorizedError(e error) bool {
	_, ok := e.(UnauthorizedError)
	return ok
}

func IsUnsupportedError(e error) bool {
	_, ok := e.(UnsupportedError)
	return ok
}

func IsNotFoundError(e error) bool {
	_, ok := e.(NotFoundError)
	return ok
}

func IsNotAllowedError(e error) bool {
	_, ok := e.(NotAllowedError)
	return ok
}

func NewBadRequestError(msg string) BadRequestError {
	return BadRequestError{msg}
}

func NewUnauthorizedError(msg string) UnauthorizedError {
	return UnauthorizedError{msg}
}

func NewNotFoundError() NotFoundError {
	return NotFoundError{i18n.T("error.not_found")}
}

func NewNotFoundMessageError(msg string) NotFoundError {
	return NotFoundError{msg}
}

func NewNotAllowedError() NotAllowedError {
	return NotAllowedError{i18n.T("error.not_allowed")}
}

func NewPermissionDeniedError(msg string) PermissionDeniedError {
	return PermissionDeniedError{msg}
}

func NewNotAllowedMessageError(msg string) NotAllowedError {
	return NotAllowedError{msg}
}

func NewUnsupportedError() UnsupportedError {
	return UnsupportedError{}
}

func NewUnsupportedMessageError(msg string) UnsupportedError {
	return UnsupportedError{msg}
}

func NewRemoteApiError(code int, msg string) RemoteApiError {
	return RemoteApiError{code, msg}
}
