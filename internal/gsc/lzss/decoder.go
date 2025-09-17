package lzss

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

var (
	ErrVocOutOfRange = errors.New("voc: out of range")
)

type Decoder struct {
	r       *bufio.Reader
	voc     []byte
	out     bytes.Buffer
	flag    byte
	counter int
	rem     int64
}

func NewDecoder(r io.Reader, voc []byte, unpackLength int64) *Decoder {
	decoder := &Decoder{
		r:   bufio.NewReader(r),
		voc: voc,
		rem: unpackLength,
	}
	decoder.out.Grow(len(voc))
	return decoder
}

func (decoder *Decoder) bit() bool { return decoder.flag&0b10000000 != 0 }

func (decoder *Decoder) Read(p []byte) (n int, err error) {
	for n < len(p) {
		if decoder.rem <= 0 {
			if n == 0 {
				return 0, io.EOF
			}
			return n, nil
		}

		if decoder.out.Len() > 0 {
			w, _ := decoder.out.Read(p[n:])
			n += w
			decoder.rem -= int64(w)
			if n == len(p) {
				return n, nil
			}
			continue
		}

		if err := decoder.readCom(); err != nil {
			return n, err
		}

		if decoder.bit() {
			// Фраза
			var buf [2]byte
			if _, err := io.ReadFull(decoder.r, buf[:]); err != nil {
				return n, err
			}
			word := uint16(buf[0])<<8 | uint16(buf[1])
			from := int(word & 0x0FFF)
			count := int(word>>12) + 3
			to := from + count

			if from < 0 || to > len(decoder.voc) {
				return n, ErrVocOutOfRange
			}

			decoder.out.Write(decoder.voc[from:to])
		} else {
			// Літерал
			b, err := decoder.r.ReadByte()
			if err != nil {
				return n, err
			}
			decoder.out.WriteByte(b)
		}
	}
	return n, nil
}

func (decoder *Decoder) readCom() error {
	if decoder.counter == 0 {
		decoder.counter = 7
		var buf [1]byte
		if _, err := decoder.r.Read(buf[:]); err != nil {
			return err
		}
		decoder.flag = buf[0]
	}
	decoder.counter--
	decoder.flag <<= 1
	return nil
}
