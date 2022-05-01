# lrpc

[![Go Reference](https://pkg.go.dev/badge/go.arsenm.dev/lrpc.svg)](https://pkg.go.dev/go.arsenm.dev/lrpc)
[![Go Report Card](https://goreportcard.com/badge/go.arsenm.dev/lrpc)](https://goreportcard.com/report/go.arsenm.dev/lrpc)

A lightweight RPC framework that aims to be as easy to use as possible, while also being as lightweight as possible. Most current RPC frameworks are bloated to the point of adding 7MB to my binary, like RPCX. That is what prompted me to create this.

---

### Channels

This RPC framework supports creating channels to transfer data from server to client. My use-case for this is to implement watch functions and transfer progress in [ITD](https://gitea.arsenm.dev/Arsen6331/itd), but it can be useful for many things.

---

### Codec

When creating a server or client, a `CodecFunc` can be provided. This `CodecFunc` is provided with an `io.ReadWriter` and returns a `Codec`, which is an interface that contains encode and decode functions with the same signature ad `json.Decoder.Decode()` and `json.Encoder.Encode()`.

This allows any codec to be used for the transfer of the data, making it easy to create clients in different languages.