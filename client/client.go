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

package client

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"

	"go.arsenm.dev/lrpc/codec"
	"go.arsenm.dev/lrpc/internal/reflectutil"
	"go.arsenm.dev/lrpc/internal/types"

	"github.com/gofrs/uuid"
)

// <= go1.17 compatibility
type any = interface{}

// Client error values
var (
	ErrReturnNotChannel = errors.New("function call returns channel but return value is not a channel type")
	ErrReturnNotPointer = errors.New("function call returns value but return value is not a pointer")
	ErrMismatchedType   = errors.New("type of channel does not match type returned by server")
)

// Client is an lrpc client
type Client struct {
	conn  io.ReadWriteCloser
	codec codec.Codec

	chMtx *sync.Mutex
	chs   map[string]chan *types.Response
}

// New creates and returns a new client
func New(conn io.ReadWriteCloser, cf codec.CodecFunc) *Client {
	out := &Client{
		conn:  conn,
		codec: cf(conn),
		chs:   map[string]chan *types.Response{},
		chMtx: &sync.Mutex{},
	}

	go out.handleConn()

	return out
}

// Call calls a method on the server
func (c *Client) Call(ctx context.Context, rcvr, method string, arg interface{}, ret interface{}) error {
	// Create new v4 UUOD
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	idStr := id.String()

	ctxDoneVal := reflect.ValueOf(ctx.Done())

	// Create new channel using the generated ID
	c.chMtx.Lock()
	c.chs[idStr] = make(chan *types.Response, 1)
	c.chMtx.Unlock()

	// Encode request using codec
	err = c.codec.Encode(types.Request{
		ID:       idStr,
		Receiver: rcvr,
		Method:   method,
		Arg:      arg,
	})
	if err != nil {
		return err
	}

	// Get response from channel
	c.chMtx.Lock()
	respCh := c.chs[idStr]
	c.chMtx.Unlock()
	resp := <-respCh

	// Close and delete channel
	c.chMtx.Lock()
	close(c.chs[idStr])
	delete(c.chs, idStr)
	c.chMtx.Unlock()

	// If response is an error, return error
	if resp.IsError {
		return errors.New(resp.Error)
	}

	// If there is no return value, stop now
	if resp.Return == nil {
		return nil
	}

	// Get reflect value of return value
	retVal := reflect.ValueOf(ret)

	// If response is a channel
	if resp.IsChannel {
		// If return value is not a channel, return error
		if retVal.Kind() != reflect.Chan {
			return ErrReturnNotChannel
		}
		// Get channel ID returned in response
		chID := resp.Return.(string)

		// Create new channel using channel ID
		c.chMtx.Lock()
		c.chs[chID] = make(chan *types.Response, 5)
		c.chMtx.Unlock()

		go func() {
			// Get type of channel elements
			chElemType := retVal.Type().Elem()
			// For every value received from channel
			for val := range c.chs[chID] {
				//s := time.Now()
				if val.ChannelDone {
					// Close and delete channel
					c.chMtx.Lock()
					close(c.chs[chID])
					delete(c.chs, chID)
					c.chMtx.Unlock()

					// Close return channel
					retVal.Close()
					break
				}
				// Get reflect value from channel response
				rVal := reflect.ValueOf(val.Return)

				// If return value is not the same as the channel
				if rVal.Type() != chElemType {
					// Attempt to convert value, skip if impossible
					newVal, err := reflectutil.Convert(rVal, chElemType)
					if err != nil {
						continue
					}
					rVal = newVal
				}

				chosen, _, _ := reflect.Select([]reflect.SelectCase{
					{Dir: reflect.SelectSend, Chan: retVal, Send: rVal},
					{Dir: reflect.SelectRecv, Chan: ctxDoneVal, Send: reflect.Value{}},
				})
				if chosen == 1 {
					c.Call(context.Background(), "lrpc", "ChannelDone", chID, nil)
					// Close and delete channel
					c.chMtx.Lock()
					close(c.chs[chID])
					delete(c.chs, chID)
					c.chMtx.Unlock()

					retVal.Close()
				}
			}
		}()
	} else {
		// IF return value is not a pointer, return error
		if retVal.Kind() != reflect.Ptr {
			return ErrReturnNotPointer
		}

		// Get return type
		retType := retVal.Type().Elem()
		// Get refkect value from response
		rVal := reflect.ValueOf(resp.Return)

		// If types do not match
		if rVal.Type() != retType {
			// Attempt to convert types, return error if not possible
			newVal, err := reflectutil.Convert(rVal, retType)
			if err != nil {
				return err
			}
			rVal = newVal
		}

		// Set return value to received value
		retVal.Elem().Set(rVal)
	}

	return nil
}

func (c *Client) handleConn() {
	for {
		resp := &types.Response{}
		// Attempt to decode response using codec
		err := c.codec.Decode(resp)
		if err != nil {
			continue
		}

		c.chMtx.Lock()
		// Get channel from map, skip if it doesn't exist
		ch, ok := c.chs[resp.ID]
		if !ok {
			c.chMtx.Unlock()
			continue
		}
		c.chMtx.Unlock()

		// Send response to channel
		ch <- resp
	}
}

// Close closes the client
func (c *Client) Close() error {
	return c.conn.Close()
}
