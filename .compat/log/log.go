package corelog

import core "dappco.re/go"

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
		return core.Sprintf("%s: %s: %v", e.Op, e.Msg, e.Err)
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

func E(op, msg string, cause any) *Err {
	err, ok := cause.(error)
	if !ok && cause != nil {
		err = core.NewError(core.Sprintf("%v", cause))
	}
	return &Err{Op: op, Msg: msg, Err: err}
}
