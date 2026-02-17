package device

type Device interface {
	ControlFunc(arguments ...Argument) Response
}

type Argument struct {
	Name  string
	Value string
}

type Response struct {
	Value string
}
