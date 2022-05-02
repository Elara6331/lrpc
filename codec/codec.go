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
