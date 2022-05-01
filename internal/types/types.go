package types

// Request represents a request sent to the server
type Request struct {
	ID       string
	Receiver string
	Method   string
	Arg      any
}

// Response represents a response returned by the server
type Response struct {
	ID        string
	ChannelDone bool
	IsChannel bool
	IsError   bool
	Error     string
	Return    any
}
