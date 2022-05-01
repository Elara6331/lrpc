package codec

import (
	"encoding/gob"
	"encoding/json"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

// CodecFunc is a function that returns a new Codec
// bound to the given io.ReadWriter
type CodecFunc func(io.ReadWriter) Codec

// Codec is able to encode and decode messages to/from
// the given io.ReadWriter
type Codec interface {
	Encode(val any) error
	Decode(val any) error
}

// Default is the default CodecFunc
var Default = Msgpack

// JSON is a CodecFunc that creates a JSON Codec
func JSON(rw io.ReadWriter) Codec {
	type jsonCodec struct {
		*json.Encoder
		*json.Decoder
	}
	return jsonCodec{
		Encoder: json.NewEncoder(rw),
		Decoder: json.NewDecoder(rw),
	}
}

// Msgpack is a CodecFunc that creates a Msgpack Codec
func Msgpack(rw io.ReadWriter) Codec {
	type msgpackCodec struct {
		*msgpack.Encoder
		*msgpack.Decoder
	}
	return msgpackCodec{
		Encoder: msgpack.NewEncoder(rw),
		Decoder: msgpack.NewDecoder(rw),
	}
}

// Gob is a CodecFunc that creates a Gob Codec
func Gob(rw io.ReadWriter) Codec {
	type gobCodec struct {
		*gob.Encoder
		*gob.Decoder
	}
	return gobCodec{
		Encoder: gob.NewEncoder(rw),
		Decoder: gob.NewDecoder(rw),
	}
}
