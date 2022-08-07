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
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

// <= go1.17 compatibility
type any = interface{}

// CodecFunc is a function that returns a new Codec
// bound to the given io.ReadWriter
type CodecFunc func(io.ReadWriter) Codec

// Codec is able to encode and decode messages to/from
// the given io.ReadWriter
type Codec interface {
	Encode(val any) error
	Decode(val any) error
	Unmarshal(data []byte, v any) error
	Marshal(v any) ([]byte, error)
}

// Default is the default CodecFunc
var Default = Msgpack

type JsonCodec struct {
	*json.Encoder
	*json.Decoder
}

func (JsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (JsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// JSON is a CodecFunc that creates a JSON Codec
func JSON(rw io.ReadWriter) Codec {
	return JsonCodec{
		Encoder: json.NewEncoder(rw),
		Decoder: json.NewDecoder(rw),
	}
}

type MsgpackCodec struct {
	*msgpack.Encoder
	*msgpack.Decoder
}

func (MsgpackCodec) Unmarshal(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

func (MsgpackCodec) Marshal(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

// Msgpack is a CodecFunc that creates a Msgpack Codec
func Msgpack(rw io.ReadWriter) Codec {
	return MsgpackCodec{
		Encoder: msgpack.NewEncoder(rw),
		Decoder: msgpack.NewDecoder(rw),
	}
}

type GobCodec struct {
	*gob.Encoder
	*gob.Decoder
}

func (GobCodec) Unmarshal(data []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}

func (GobCodec) Marshal(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := gob.NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Gob is a CodecFunc that creates a Gob Codec
func Gob(rw io.ReadWriter) Codec {
	return GobCodec{
		Encoder: gob.NewEncoder(rw),
		Decoder: gob.NewDecoder(rw),
	}
}
