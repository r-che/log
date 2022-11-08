package log

import "fmt"

type OpError struct {
	err error
}
func (e *OpError) Error() string {
	return e.err.Error()
}

type FileError struct {
	OpError
	fileErr	error
}
func (ef *FileError) Unwrap() error {
	return ef.fileErr
}
func NewFileError(format string, err error) error {
	return &FileError{OpError{fmt.Errorf(format, err)}, err}
}
