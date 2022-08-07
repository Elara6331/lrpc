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

package types

// <= go1.17 compatibility
type any = interface{}

// Request represents a request sent to the server
type Request struct {
	ID       string
	Receiver string
	Method   string
	Arg      []byte
}

type ResponseType uint8

const (
	ResponseTypeNormal ResponseType = iota
	ResponseTypeError
	ResponseTypeChannel
	ResponseTypeChannelDone
)

// Response represents a response returned by the server
type Response struct {
	Type   ResponseType
	ID     string
	Error  string
	Return []byte
}
