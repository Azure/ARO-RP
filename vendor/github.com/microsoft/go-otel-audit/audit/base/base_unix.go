//go:build unix

package base

import (
	"bytes"
	"unsafe"

	"golang.org/x/sys/unix"
)

func kernelVer() (string, error) {
	uts, err := uname()
	if err != nil {
		return "", err
	}
	trimRelease := bytes.Trim(uts.Release[:], "\x00")
	return bytesToStr(trimRelease), nil
}

func uname() (unix.Utsname, error) {
	uts := unix.Utsname{}

	if err := unix.Uname(&uts); err != nil {
		return uts, err
	}
	return uts, nil
}

// bytesToStr converts a byte slice to a string without a copy (aka no allocation).
// This uses unsafe, so only use in cases where you know the byte slice is not going to be modified.
func bytesToStr(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
