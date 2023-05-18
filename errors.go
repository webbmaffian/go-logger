package logger

import "errors"

var (
	ErrInvalidCertificate  = errors.New("invalid certificate")
	ErrInvalidSerialNumber = errors.New("invalid serial number")
	ErrInvalidSubjectKeyId = errors.New("invalid subject key ID")
	ErrFull                = errors.New("full")
	ErrEmpty               = errors.New("empty")
	ErrTooShort            = errors.New("entry too short")
	ErrTooLong             = errors.New("entry too long")
	ErrInvalidSeverity     = errors.New("invalid severity")
	ErrTooManyTags         = errors.New("too many tags")
	ErrTooManyMetric       = errors.New("too many metric key/value pairs")
	ErrTooManyMeta         = errors.New("too many meta key/value pairs")
	ErrTooManyStackTrace   = errors.New("too many stack traces")
	ErrCorruptEntry        = errors.New("corrupt entry")
	ErrForbiddenBucket     = errors.New("forbidden bucket")
	ErrNaN                 = errors.New("not a number")
)
