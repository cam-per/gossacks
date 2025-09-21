package lzstd

import (
	"bufio"
	"bytes"
	"errors"
	"io"

	"gitgub.com/cam-per/gossacks/utils"
)

var (
	ErrVocOutOfRange = errors.New("voc: out of range")
)

type Decoder struct {
	r         *bufio.Reader
	voc       []byte
	out       bytes.Buffer
	flag      byte
	bitsLeft  int
	remaining int64
}

func NewDecoder(r io.Reader, voc []byte, unpackLength int64) *Decoder {
	decoder := &Decoder{
		r:         bufio.NewReader(r),
		voc:       voc,
		remaining: unpackLength,
		bitsLeft:  0,
	}
	decoder.out.Grow(len(voc))
	return decoder
}

func (decoder *Decoder) Read(p []byte) (n int, err error) {

	for decoder.remaining > 0 || decoder.out.Len() > 0 {
		nn, err := decoder.flushBuffer(p[n:])
		if err != nil {
			return n, err
		}
		n += nn
		if n == len(p) {
			return n, nil
		}

		if decoder.remaining <= 0 {
			continue
		}

		if decoder.bitsLeft == 0 {
			decoder.flag, err = utils.ReadByte(decoder.r)
			if err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return n, err
			}
			decoder.bitsLeft = 8
		}

		if (decoder.flag & 0x80) != 0 {
			word, err := utils.ReadUint16LE(decoder.r)
			if err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return n, err
			}

			count := int((word >> 12) + 3)
			offset := int(word & 0x0FFF)

			if offset+count > len(decoder.voc) {
				return n, ErrVocOutOfRange
			}
			decoder.out.Write(decoder.voc[offset : offset+count])
			decoder.remaining -= int64(count)
		} else {
			d, err := utils.ReadByte(decoder.r)
			if err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return n, err
			}
			decoder.out.WriteByte(d)
			decoder.remaining--
		}

		decoder.flag <<= 1
		decoder.bitsLeft--
	}
	if decoder.remaining == 0 && decoder.out.Len() == 0 {
		return 0, io.EOF
	}
	return
}

func (decoder *Decoder) flushBuffer(buf []byte) (n int, err error) {
	if decoder.out.Len() == 0 {
		return 0, nil
	}
	return decoder.out.Read(buf)
}
