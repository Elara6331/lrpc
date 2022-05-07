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

package reflectutil

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// <= go1.17 compatibility
type any = interface{}

// Convert attempts to convert the given value to the given type
func Convert(in reflect.Value, toType reflect.Type) (reflect.Value, error) {
	// Get input type
	inType := in.Type()

	// If input is already the desired type, return
	if inType == toType {
		return in, nil
	}

	// If the output type is a pointer to the input type
	if reflect.PtrTo(inType) == toType {
		if in.CanAddr() {
			// Return pointer to input
			return in.Addr(), nil
		}

		inPtrVal := reflect.New(inType)
		inPtrVal.Elem().Set(in)
		return inPtrVal, nil
	}

	// If input is a pointer pointing to the output type
	if inType.Kind() == reflect.Ptr && inType.Elem() == toType {
		// Return value being pointed at by input
		return reflect.Indirect(in), nil
	}

	// If input can be converted to desired type, convert and return
	if in.CanConvert(toType) {
		return in.Convert(toType), nil
	}

	// Create new value of desired type
	to := reflect.New(toType).Elem()

	// If type is a pointer
	if to.Kind() == reflect.Ptr {
		// Initialize value
		to.Set(reflect.New(to.Type().Elem()))
	}

	switch val := in.Interface().(type) {
	case string:
		// If desired type satisfies text unmarshaler
		if u, ok := to.Interface().(encoding.TextUnmarshaler); ok {
			// Use text unmarshaler to get value
			err := u.UnmarshalText([]byte(val))
			if err != nil {
				return reflect.Value{}, err
			}

			// Return unmarshaled value
			return reflect.ValueOf(any(u)), nil
		}
	case []byte:
		// If desired type satisfies binary unmarshaler
		if u, ok := to.Interface().(encoding.BinaryUnmarshaler); ok {
			// Use binary unmarshaler to get value
			err := u.UnmarshalBinary(val)
			if err != nil {
				return reflect.Value{}, err
			}

			// Return unmarshaled value
			return reflect.ValueOf(any(u)), nil
		}
	}

	// If input is a map
	if in.Kind() == reflect.Map {
		// Use mapstructure to decode value
		err := mapstructure.Decode(in.Interface(), to.Addr().Interface())
		if err == nil {
			return to, nil
		}
	}

	// If input is a slice of any, and output is an array or slice
	if in.Type() == reflect.TypeOf([]any{}) &&
		to.Kind() == reflect.Slice || to.Kind() == reflect.Array {
		// Use ConvertSlice to convert value
		return reflect.ValueOf(ConvertSlice(
			in.Interface().([]any),
			toType,
		)), nil
	}

	return to, fmt.Errorf("cannot convert %s to %s", inType, toType)
}

// ConvertSlice converts []any to an array or slice, as provided
// in the "to" argument.
func ConvertSlice(in []any, to reflect.Type) any {
	// Create new value for output
	out := reflect.New(to).Elem()

	// If output value is a slice
	if out.Kind() == reflect.Slice {
		// Get type of slice elements
		outType := out.Type().Elem()

		// For every value provided
		for i := 0; i < len(in); i++ {
			// Get value of input type
			inVal := reflect.ValueOf(in[i])
			// Create new output type
			outVal := reflect.New(outType).Elem()

			// If types match
			if inVal.Type() == outType {
				// Set output value to input value
				outVal.Set(inVal)
			} else {
				newVal, err := Convert(inVal, outType)
				if err != nil {
					// Set output value to its zero value
					outVal.Set(reflect.Zero(outVal.Type()))
				} else {
					outVal.Set(newVal)
				}
			}

			// Append output value to slice
			out = reflect.Append(out, outVal)
		}
	} else if out.Kind() == reflect.Array && out.Len() == len(in) {
		//If output type is array and lengths match

		// For every input value
		for i := 0; i < len(in); i++ {
			// Get matching output index
			outVal := out.Index(i)
			// Get input value
			inVal := reflect.ValueOf(in[i])

			// If types match
			if inVal.Type() == outVal.Type() {
				// Set output value to input value
				outVal.Set(inVal)
			} else {
				// If input value can be converted to output type
				if inVal.CanConvert(outVal.Type()) {
					// Convert and set output value to input value
					outVal.Set(inVal.Convert(outVal.Type()))
				} else {
					// Set output value to its zero value
					outVal.Set(reflect.Zero(outVal.Type()))
				}
			}
		}
	}

	// Return created value
	return out.Interface()
}
