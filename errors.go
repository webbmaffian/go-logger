package logger

import "errors"

var (
	ErrInvalidCertificate  = errors.New("invalid certificate")
	ErrInvalidSerialNumber = errors.New("invalid serial number")
	ErrInvalidSubjectKeyId = errors.New("invalid subject key ID")
	ErrTooShort            = errors.New("entry too short")
	ErrCorruptEntry        = errors.New("corrupt entry")
	ErrForbiddenBucket     = errors.New("forbidden bucket")
)
