//go:build !go1.20
// +build !go1.20

package logger

import (
	"reflect"
	"unsafe"
)

// Converts byte slice to a string without memory allocation.
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Converts string to a byte slice without memory allocation.
func s2b(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	return b
}
