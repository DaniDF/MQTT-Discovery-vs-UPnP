package device

type Device interface {
	String() string
	Name() string
}

type Argument struct {
	Name  string
	Value string
}

type Response struct {
	Value        string
	ErrorCode    int // Value different from 0 will be considered errors (in that case Value is ignored)
	ErrorMessage string
}
