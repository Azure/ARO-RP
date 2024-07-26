package mimo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "fmt"

type MIMOErrorVariety string

const (
	MIMOErrorTypeTransientError MIMOErrorVariety = "TransientError"
	MIMOErrorTypeTerminalError  MIMOErrorVariety = "TerminalError"
)

type MIMOError interface {
	error
	MIMOErrorVariety() MIMOErrorVariety
}

type wrappedMIMOError struct {
	error
	variety MIMOErrorVariety
}

func (f wrappedMIMOError) MIMOErrorVariety() MIMOErrorVariety {
	return f.variety
}

func (f wrappedMIMOError) Error() string {
	return fmt.Sprintf("%s: %s", f.variety, f.error.Error())
}

func NewMIMOError(err error, variety MIMOErrorVariety) MIMOError {
	return wrappedMIMOError{
		error:   err,
		variety: variety,
	}
}

func TerminalError(err error) MIMOError {
	return NewMIMOError(err, MIMOErrorTypeTerminalError)
}

func TransientError(err error) MIMOError {
	return NewMIMOError(err, MIMOErrorTypeTransientError)
}

func IsRetryableError(err error) bool {
	e, ok := err.(wrappedMIMOError)
	if !ok {
		return false
	}
	if e.MIMOErrorVariety() == MIMOErrorTypeTransientError {
		return true
	}
	return false
}
