/*
 *	lrpc allows for clients to call functions on a server remotely.
 *	Copyright (C) 2022 Arsen Musayelyan
 *
 *	This program is free software: you can redistribute it and/or modify
 *	it under the terms of the GNU General Public License as published by
 *	the Free Software Foundation, either version 3 of the License, or
 *	(at your option) any later version.
 *
 *	This program is distributed in the hope that it will be useful,
 *	but WITHOUT ANY WARRANTY; without even the implied warranty of
 *	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *	GNU General Public License for more details.
 *
 *	You should have received a copy of the GNU General Public License
 *	along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"sync"

	"go.arsenm.dev/lrpc/codec"
	"go.arsenm.dev/lrpc/internal/reflectutil"
	"go.arsenm.dev/lrpc/internal/types"
	"golang.org/x/net/websocket"
)

// <= go1.17 compatibility
type any = interface{}

var (
	ErrInvalidType        = errors.New("type must be struct or pointer to struct")
	ErrNoSuchReceiver     = errors.New("no such receiver registered")
	ErrNoSuchMethod       = errors.New("no such method was found")
	ErrInvalidMethod      = errors.New("method invalid for lrpc call")
	ErrUnexpectedArgument = errors.New("argument provided but the function does not accept any arguments")
)

// Server is an lrpc server
type Server struct {
	rcvrs map[string]reflect.Value

	contextsMtx sync.Mutex
	contexts    map[string]*Context
}

// New creates and returns a new server
func New() *Server {
	// Create new server
	out := &Server{
		rcvrs:    map[string]reflect.Value{},
		contexts: map[string]*Context{},
	}

	// Register lrpc functions
	out.Register(lrpc{out})

	return out
}

// Close closes the server
func (s *Server) Close() {
	for _, ctx := range s.contexts {
		ctx.cancel()
	}
}

// Register registers a value to be called by a client
func (s *Server) Register(v any) error {
	// Get reflect values for v
	val := reflect.ValueOf(v)
	kind := val.Kind()

	// create variable to store name of v
	var name string
	switch kind {
	case reflect.Ptr:
		// If v is a pointer, get the name of the underlying type
		name = val.Elem().Type().Name()
	case reflect.Struct:
		// If v is a struct, get its name
		name = val.Type().Name()
	default:
		// If v is not pointer or struct, return error
		return ErrInvalidType
	}

	// Add v to receivers map
	s.rcvrs[name] = val

	return nil
}

// execute runs a method of a registered value
func (s *Server) execute(pCtx context.Context, typ string, name string, arg any, c codec.Codec) (a any, ctx *Context, err error) {
	// Try to get value from receivers map
	val, ok := s.rcvrs[typ]
	if !ok {
		return nil, nil, ErrNoSuchReceiver
	}

	// Try to retrieve given method
	mtd := val.MethodByName(name)
	if !mtd.IsValid() {
		return nil, nil, ErrNoSuchMethod
	}

	// If method invalid, return error
	if !mtdValid(mtd) {
		return nil, nil, ErrInvalidMethod
	}

	// Get method type
	mtdType := mtd.Type()

	// Return error if argument provided but isn't expected
	if mtdType.NumIn() == 1 && arg != nil {
		return nil, nil, ErrUnexpectedArgument
	}

	// IF argument is []any
	anySlice, ok := arg.([]any)
	if ok {
		// Convert slice to the method's arg type and
		// set arg to the newly-converted slice
		arg = reflectutil.ConvertSlice(anySlice, mtdType.In(1))
	}

	// Get argument value
	argVal := reflect.ValueOf(arg)
	// If argument's type does not match method's argument type
	if arg != nil && argVal.Type() != mtdType.In(1) {
		val, err = reflectutil.Convert(argVal, mtdType.In(1))
		if err != nil {
			return nil, nil, err
		}
		arg = val.Interface()
	}

	ctx = newContext(pCtx, c)
	// Get reflect value of context
	ctxVal := reflect.ValueOf(ctx)

	switch mtdType.NumOut() {
	case 0: // If method has no return values
		if mtdType.NumIn() == 2 {
			// Call method with arg, ignore returned value
			mtd.Call([]reflect.Value{ctxVal, reflect.ValueOf(arg)})
		} else {
			// Call method without arg, ignore returned value
			mtd.Call([]reflect.Value{ctxVal})
		}
	case 1: // If method has one return value
		if mtdType.NumIn() == 2 {
			// Call method with arg, get returned values
			out := mtd.Call([]reflect.Value{ctxVal, reflect.ValueOf(arg)})

			// If the first return value's type is error
			if mtdType.Out(0).Name() == "error" {
				// Get first return value as interface
				out0 := out[0].Interface()
				if out0 == nil {
					a, err = nil, nil
				} else {
					a, err = nil, out0.(error)
				}
			} else {
				a, err = out[0].Interface(), nil
			}
		} else {
			// Call method without arg, get returned values
			out := mtd.Call([]reflect.Value{ctxVal})

			// If the first return value's type is error
			if mtdType.Out(0).Name() == "error" {
				// Get first return value as interface
				out0 := out[0].Interface()
				if out0 == nil {
					a, err = nil, nil
				} else {
					a, err = nil, out0.(error)
				}
			} else {
				a, err = out[0].Interface(), nil
			}
		}
	case 2: // If method has two return values
		if mtdType.NumIn() == 2 {
			// Call method with arg and get returned values
			out := mtd.Call([]reflect.Value{ctxVal, reflect.ValueOf(arg)})

			// Get second return value as interface
			out1 := out[1].Interface()
			if out1 != nil {
				err, ok = out1.(error)

				// If second return value is not an error, the function is invalid
				if !ok {
					a, err = nil, ErrInvalidMethod
				}
			}

			a = out[0].Interface()
		} else {
			// Call method without arg and get returned values
			out := mtd.Call([]reflect.Value{ctxVal})

			// Get second return value as interface
			out1 := out[1].Interface()
			if out1 != nil {

				// If second return value is not an error, the function is invalid
				err, ok = out1.(error)
				if !ok {
					a, err = nil, ErrInvalidMethod
				}
			}

			a = out[0].Interface()
		}
	}

	return a, ctx, err
}

// Serve starts the server using the provided listener
// and codec function
func (s *Server) Serve(ctx context.Context, ln net.Listener, cf codec.CodecFunc) {
	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if errors.Is(err, net.ErrClosed) {
			break
		} else if err != nil {
			continue
		}

		// Create new instance of codec bound to conn
		c := cf(conn)
		// Handle connection
		go s.handleConn(ctx, c)
	}
}

// ServeWS starts a server using WebSocket. This may be useful for
// clients written in other languages, such as JS for a browser.
func (s *Server) ServeWS(ctx context.Context, addr string, cf codec.CodecFunc) (err error) {
	// Create new WebSocket server
	ws := websocket.Server{}

	// Create new WebSocket config
	ws.Config = websocket.Config{
		Version: websocket.ProtocolVersionHybi13,
	}

	// Set server handler
	ws.Handler = func(c *websocket.Conn) {
		s.handleConn(c.Request().Context(), cf(c))
	}

	server := &http.Server{
		Addr: addr,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
		Handler: http.HandlerFunc(ws.ServeHTTP),
	}

	// Listen and serve on given address
	return server.ListenAndServe()
}

// ServeConn uses the provided connection to serve the client.
// This may be useful if something other than a net.Listener
// needs to be used
func (s *Server) ServeConn(ctx context.Context, conn io.ReadWriter, cf codec.CodecFunc) {
	s.handleConn(ctx, cf(conn))
}

// handleConn handles a connection
func (s *Server) handleConn(pCtx context.Context, c codec.Codec) {
	codecMtx := &sync.Mutex{}

	for {
		var call types.Request
		// Read request using codec
		err := c.Decode(&call)
		if err == io.EOF {
			break
		} else if err != nil {
			s.sendErr(c, call, nil, err)
			continue
		}

		// Execute decoded call
		val, ctx, err := s.execute(
			pCtx,
			call.Receiver,
			call.Method,
			call.Arg,
			c,
		)
		if err != nil {
			s.sendErr(c, call, val, err)
		} else {
			// Create response
			res := types.Response{
				ID:     call.ID,
				Return: val,
			}

			// If function has created a channel
			if ctx.isChannel {
				// Set IsChannel to true
				res.Type = types.ResponseTypeChannel
				// Overwrite return value with channel ID
				res.Return = ctx.channelID

				// Store context in map for future use
				s.contextsMtx.Lock()
				s.contexts[ctx.channelID] = ctx
				s.contextsMtx.Unlock()

				go func() {
					// For every value received from channel
					for val := range ctx.channel {
						codecMtx.Lock()
						// Encode response using codec
						c.Encode(types.Response{
							ID:     ctx.channelID,
							Return: val,
						})
						codecMtx.Unlock()
					}

					// Cancel context
					ctx.cancel()
					// Delete context from map
					s.contextsMtx.Lock()
					delete(s.contexts, ctx.channelID)
					s.contextsMtx.Unlock()

					codecMtx.Lock()
					c.Encode(types.Response{
						Type: types.ResponseTypeChannelDone,
						ID:   ctx.channelID,
					})
					codecMtx.Unlock()
				}()
			}

			// Encode response using codec
			codecMtx.Lock()
			c.Encode(res)
			codecMtx.Unlock()
		}
	}
}

// sendErr sends an error response
func (s *Server) sendErr(c codec.Codec, req types.Request, val any, err error) {
	// Encode error response using codec
	c.Encode(types.Response{
		Type:   types.ResponseTypeError,
		ID:     req.ID,
		Error:  err.Error(),
		Return: val,
	})
}

// lrpc contains functions registered on every server
type lrpc struct {
	srv *Server
}

// ChannelDone cancels a context and closes the associated channel
func (l lrpc) ChannelDone(_ *Context, id string) {
	// Try to get context
	ctx, ok := l.srv.contexts[id]
	if !ok {
		return
	}

	// Cancel context
	ctx.cancel()
	// Delete context from map
	l.srv.contextsMtx.Lock()
	delete(l.srv.contexts, id)
	l.srv.contextsMtx.Unlock()
}

// MethodDesc describes methods on a receiver
type MethodDesc struct {
	Name    string
	Args    []string
	Returns []string
}

// Introspect returns method descriptions for the given receiver
func (l lrpc) Introspect(_ *Context, name string) ([]MethodDesc, error) {
	// Attempt to get receiver
	rcvr, ok := l.srv.rcvrs[name]
	if !ok {
		return nil, ErrNoSuchReceiver
	}
	// Get receiver type value
	rcvrType := rcvr.Type()

	// Create slice for output
	var out []MethodDesc
	// For every method on receiver
	for i := 0; i < rcvr.NumMethod(); i++ {
		// Get receiver method
		mtd := rcvr.Method(i)
		// If invalid, skip
		if !mtdValid(mtd) {
			continue
		}
		// Get method type
		mtdType := mtd.Type()

		// Get amount of arguments
		numIn := mtdType.NumIn()
		args := make([]string, numIn-1)
		// For every argument, store type in slice
		// Skip first argument, as it is *Context
		for i := 1; i < numIn; i++ {
			args[i-1] = mtdType.In(i).String()
		}

		// Get amount of returns
		numOut := mtdType.NumOut()
		returns := make([]string, numOut)
		// For every return, store type in slice
		for i := 0; i < numOut; i++ {
			returns[i] = mtdType.Out(i).String()
		}

		out = append(out, MethodDesc{
			Name:    rcvrType.Method(i).Name,
			Args:    args,
			Returns: returns,
		})
	}

	return out, nil
}

// IntrospectAll runs Introspect on all registered receivers and returns all results
func (l lrpc) IntrospectAll(_ *Context) (map[string][]MethodDesc, error) {
	// Create map for output
	out := make(map[string][]MethodDesc, len(l.srv.rcvrs))
	// For every registered receiver
	for name := range l.srv.rcvrs {
		// Introspect receiver
		descs, err := l.Introspect(nil, name)
		if err != nil {
			return nil, err
		}

		// Set results in map
		out[name] = descs
	}
	return out, nil
}

func mtdValid(mtd reflect.Value) bool {
	// Get method's type
	mtdType := mtd.Type()

	// If method has more than 2 or less than 1 input, it is invalid
	if mtdType.NumIn() > 2 || mtdType.NumIn() < 1 {
		return false
	}

	// If method has more than 2 outputs, it is invalid
	if mtdType.NumOut() > 2 {
		return false
	}

	// Check to ensure first parameter is context
	if mtdType.In(0) != reflect.TypeOf((*Context)(nil)) {
		return false
	}

	// If method has 2 outputs
	if mtdType.NumOut() == 2 {
		// Check to ensure the second one is an error
		if mtdType.Out(1).Name() != "error" {
			return false
		}
	}

	return true
}
