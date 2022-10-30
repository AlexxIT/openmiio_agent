package rpc

type Request func(from int, req *Message) bool
type Response func(to int, req Message, res *Message) bool

var Requests []Request
var Responses []Response

func AddRequest(f Request) {
	Requests = append(Requests, f)
}

func AddResponse(f Response) {
	Responses = append(Responses, f)
}
