package main

import (
	"context"
	"encoding/gob"
	"net"

	"go.arsenm.dev/lrpc/codec"
	"go.arsenm.dev/lrpc/server"
)

type Arith struct{}

func (Arith) Add(ctx *server.Context, in [2]int) int {
	return in[0] + in[1]
}

func (Arith) Mul(ctx *server.Context, in [2]int) int {
	return in[0] * in[1]
}

func (Arith) Div(ctx *server.Context, in [2]int) int {
	return in[0] / in[1]
}

func (Arith) Sub(ctx *server.Context, in [2]int) int {
	return in[0] - in[1]
}

func main() {
	gob.Register([2]int{})

	s := server.New()
	s.Register(Arith{})

	ln, _ := net.Listen("tcp", ":9090")
	s.Serve(context.Background(), ln, codec.Gob)
}
