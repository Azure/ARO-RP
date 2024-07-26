package mimo

type MIMOErrorVariety int

const (
	MIMOErrorTypeTransientError MIMOErrorVariety = iota
	MIMOErrorTypeNonRetryableError
)

type MIMOError interface {
	error
	MIMOErrorVariety() MIMOErrorVariety
}

type wrappedMIMOError struct {
	error
	variety MIMOErrorVariety
}

func (f *wrappedMIMOError) MIMOErrorVariety() MIMOErrorVariety {
	return f.variety
}

func NewMIMOError(err error, variety MIMOErrorVariety) MIMOError {
	return &wrappedMIMOError{
		error:   err,
		variety: variety,
	}
}

func UnretryableError(err error) MIMOError {
	return NewMIMOError(err, MIMOErrorTypeNonRetryableError)
}

func TransientError(err error) MIMOError {
	return NewMIMOError(err, MIMOErrorTypeTransientError)
}
