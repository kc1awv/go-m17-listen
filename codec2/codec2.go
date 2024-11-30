/*
Copyright (C) 2024 Steve Miller KC1AWV

This program is free software: you can redistribute it and/or modify it
under the terms of the GNU General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option)
any later version.

This program is distributed in the hope that it will be useful, but WITHOUT
ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
more details.

You should have received a copy of the GNU General Public License along with
this program. If not, see <http://www.gnu.org/licenses/>.
*/

package codec2

/*
#cgo LDFLAGS: -lcodec2
#include <codec2/codec2.h>
#include <stdlib.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

type Codec2 struct {
	handle *C.struct_CODEC2
	mode   int
}

const (
	MODE_3200 = C.CODEC2_MODE_3200
)

func New(mode int) (*Codec2, error) {
	handle := C.codec2_create(C.int(mode))
	if handle == nil {
		return nil, errors.New("failed to create codec2")
	}
	return &Codec2{handle: handle, mode: mode}, nil
}

func (c *Codec2) Close() {
	C.codec2_destroy(c.handle)
}

func (c *Codec2) Decode(bits []byte) ([]int16, error) {
	nsam := C.codec2_samples_per_frame(c.handle)
	nbit := C.codec2_bits_per_frame(c.handle)

	if len(bits) != int(nbit/8) {
		return nil, errors.New("invalid bit length")
	}

	audio := make([]int16, nsam)
	C.codec2_decode(c.handle, (*C.short)(unsafe.Pointer(&audio[0])), (*C.uchar)(unsafe.Pointer(&bits[0])))

	return audio, nil
}
