package log

import "fmt"

type Err struct {
	Op  string
	Msg string
	Err error
}

func (e *Err) Error() string {
	if e == nil {
		return ""
	}
	switch {
	case e.Op != "" && e.Msg != "" && e.Err != nil:
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Msg, e.Err)
	case e.Op != "" && e.Msg != "":
		return e.Op + ": " + e.Msg
	case e.Msg != "" && e.Err != nil:
		return e.Msg + ": " + e.Err.Error()
	case e.Msg != "":
		return e.Msg
	case e.Err != nil:
		return e.Err.Error()
	default:
		return e.Op
	}
}

func (e *Err) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func E(op, msg string, err error) error {
	return &Err{Op: op, Msg: msg, Err: err}
}
