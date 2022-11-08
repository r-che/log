package log

import "fmt"

type LogErr struct {
	err error
}
func (e *LogErr) Error() string {
	return e.err.Error()
}

type ErrFile struct {
	LogErr
	fileErr	error
}
func (ef *ErrFile) Unwrap() error {
	return ef.fileErr
}
func NewErrFile(format string, err error) error {
	return &ErrFile{LogErr{fmt.Errorf(format, err)}, err}
}
