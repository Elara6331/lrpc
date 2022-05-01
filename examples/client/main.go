package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"

	"go.arsenm.dev/lrpc/client"
	"go.arsenm.dev/lrpc/codec"
)

func main() {
	gob.Register([2]int{})

	ctx := context.Background()

	conn, _ := net.Dial("tcp", "localhost:9090")
	c := client.New(conn, codec.Gob)
	defer c.Close()

	var add int
	c.Call(ctx, "Arith", "Add", [2]int{5, 5}, &add)

	var sub int
	c.Call(ctx, "Arith", "Sub", [2]int{5, 5}, &sub)

	var mul int
	c.Call(ctx, "Arith", "Mul", [2]int{5, 5}, &mul)

	var div int
	c.Call(ctx, "Arith", "Div", [2]int{5, 5}, &div)

	fmt.Printf(
		"add: %d, sub: %d, mul: %d, div: %d\n",
		add, sub, mul, div,
	)
}
