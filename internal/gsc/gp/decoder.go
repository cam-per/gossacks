package gp

import (
	"encoding/binary"
	"io"
	"math"
)

type Reader interface {
	io.Reader
	io.ReaderAt
}

type Decoder struct {
	r          Reader
	header     header
	dataOffset uint32
	frames     []Frame
	voc        []byte
}

func NewDecoder(r Reader) *Decoder {
	return &Decoder{r: r}
}

func (decoder *Decoder) Decode() error {
	if err := binary.Read(decoder.r, binary.LittleEndian, &decoder.header); err != nil {
		return err
	}

	if err := decoder.decodeVoc(); err != nil {
		return err
	}

	frames := make([]uint32, decoder.header.PicturesCount)
	if err := binary.Read(decoder.r, binary.LittleEndian, &frames); err != nil {
		return err
	}

	decoder.frames = make([]Frame, decoder.header.PicturesCount)
	for i := range decoder.frames {
		if err := decoder.decodeFrameHeader(frames[i], &decoder.frames[i].header); err != nil {
			return err
		}
	}
	return nil
}

func (decoder *Decoder) decodeFrameHeader(offset uint32, h *frameHeader) error {
	r := io.NewSectionReader(decoder.r, int64(offset), math.MaxInt64)
	return binary.Read(r, binary.LittleEndian, h)
}

func (decoder *Decoder) decodeVoc() error {
	r := io.NewSectionReader(decoder.r, int64(decoder.header.VocOffset), int64(decoder.header.VocLength))
	buf, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	decoder.voc = buf
	return nil
}
