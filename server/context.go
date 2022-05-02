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
	"time"

	"go.arsenm.dev/lrpc/codec"

	"github.com/gofrs/uuid"
)

// Context is a connection context for RPC calls
type Context struct {
	isChannel bool
	channelID string
	channel   chan any

	codec codec.Codec

	doneCh   chan struct{}
	canceled bool
}

// MakeChannel changes the function it's called in into a
// channel function, and returns a channel which can be used
// to send information to the client.
//
// This will ovewrite any return value of the function with
// a channel ID.
func (ctx *Context) MakeChannel() (chan<- any, error) {
	ctx.isChannel = true
	chID, err := uuid.NewV4()
	ctx.channelID = chID.String()
	ctx.channel = make(chan any, 5)
	return ctx.channel, err
}

// GetCodec returns a codec bound to the connection
// that called this function
func (ctx *Context) GetCodec() codec.Codec {
	return ctx.codec
}

// Deadline always returns the current time and false
// as this context does not support deadlines
func (ctx *Context) Deadline() (time.Time, bool) {
	return time.Now(), false
}

// Value always returns nil as this context stores no values
func (ctx *Context) Value(_ any) any {
	return nil
}

// Err returns context.Canceled if the context was canceled,
// otherwise nil
func (ctx *Context) Err() error {
	if ctx.canceled {
		return context.Canceled
	}
	return nil
}

// Done returns a channel that will be closed when
// the context is canceled, such as when ChannelDone
// is called by the client
func (ctx *Context) Done() <-chan struct{} {
	return ctx.doneCh
}

// Cancel cancels the context
func (ctx *Context) Cancel() {
	ctx.canceled = true
	close(ctx.doneCh)
}
