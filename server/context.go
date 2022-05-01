package server

import (
	"go.arsenm.dev/lrpc/codec"

	"github.com/gofrs/uuid"
)

// Context is a connection context for RPC calls
type Context struct {
	isChannel bool
	channelID string
	channel   chan any

	codec codec.Codec

	doneCh chan struct{}
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

// Done returns a channel that will be closed when
// the context is canceled, such as when ChannelDone
// is called by the client
func (ctx *Context) Done() <-chan struct{} {
	return ctx.doneCh
}

// Cancel cancels the context
func (ctx *Context) Cancel() {
	close(ctx.doneCh)
}
