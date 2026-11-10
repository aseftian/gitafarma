package domain

import "errors"

var (
	ErrNotFound           = errors.New("resource not found")
	ErrAlreadyExists      = errors.New("resource already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrEmailNotVerified   = errors.New("google email is not verified")
	ErrAdminNotAllowlist  = errors.New("admin email is not in allowlist or inactive")
	ErrSheetNotConfigured = errors.New("google spreadsheet is not configured")
)
