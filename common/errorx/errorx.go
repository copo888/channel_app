package errorx

type Err struct {
	code    string
	message string
}

func New(code string, msgs ...string) error {
	e := &Err{
		code: code,
	}
	if len(msgs) > 0 {
		e.message = msgs[0]
	}
	return e
}

func (e *Err) Error() string {
	return e.code
}

func (e *Err) GetMessage() string {
	return e.message
}
