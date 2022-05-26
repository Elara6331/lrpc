package lrpc_test

import (
	"context"
	"encoding/gob"
	"net"
	"testing"

	"go.arsenm.dev/lrpc/client"
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

func TestCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create new network pipe
	sConn, cConn := net.Pipe()

	s := server.New()
	defer s.Close()
	// Register Arith for RPC
	s.Register(Arith{})
	// Serve the pipe connection using default codec
	go s.ServeConn(ctx, sConn, codec.Default)

	// Create new client using default codec
	c := client.New(cConn, codec.Default)
	defer c.Close()

	// Call Arith.Add()
	var add int
	err := c.Call(ctx, "Arith", "Add", [2]int{5, 5}, &add)
	if err != nil {
		t.Error(err)
	}

	// Call Arith.Sub()
	var sub int
	err = c.Call(ctx, "Arith", "Sub", [2]int{5, 5}, &sub)
	if err != nil {
		t.Error(err)
	}

	// Call Arith.Mul()
	var mul int
	err = c.Call(ctx, "Arith", "Mul", [2]int{5, 5}, &mul)
	if err != nil {
		t.Error(err)
	}

	// Call Arith.Div()
	var div int
	err = c.Call(ctx, "Arith", "Div", [2]int{5, 5}, &div)
	if err != nil {
		t.Error(err)
	}

	if add != 10 {
		t.Errorf("add: expected 10, got %d", add)
	}

	if sub != 0 {
		t.Errorf("sub: expected 0, got %d", sub)
	}

	if mul != 25 {
		t.Errorf("mul: expected 25, got %d", mul)
	}

	if div != 1 {
		t.Errorf("div: expected 1, got %d", div)
	}
}

func TestCodecs(t *testing.T) {
	// Register the 2-integer array for gob
	gob.Register([2]int{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create function to test each codec
	testCodec := func(cf codec.CodecFunc, name string) {
		// Create network pipe
		sConn, cConn := net.Pipe()

		s := server.New()
		defer s.Close()
		// Register Arith for RPC
		s.Register(Arith{})
		// Serve the pipe connection using provided codec
		go s.ServeConn(ctx, sConn, cf)

		// Create new client using provided codec
		c := client.New(cConn, cf)
		defer c.Close()

		// Call Arith.Add()
		var add int
		err := c.Call(ctx, "Arith", "Add", [2]int{2, 2}, &add)
		if err != nil {
			t.Errorf("codec/%s: %v", name, err)
		}

		if add != 4 {
			t.Errorf("codec/%s: add: expected 4, got %d", name, add)
		}
	}

	// Test all codecs
	testCodec(codec.Msgpack, "msgpack")
	testCodec(codec.JSON, "json")
	testCodec(codec.Gob, "gob")
}
