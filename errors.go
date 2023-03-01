package logger

import "errors"

var (
	ErrInvalidCertificate  = errors.New("invalid certificate")
	ErrInvalidSerialNumber = errors.New("invalid serial number")
	ErrInvalidSubjectKeyId = errors.New("invalid subject key ID")
	ErrFull                = errors.New("full")
	ErrEmpty               = errors.New("empty")
)
