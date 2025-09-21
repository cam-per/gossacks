package gp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"image/color"
	"io"

	"gitgub.com/cam-per/gossacks/gsc/lzstd"
)

type Decoder struct {
	header  header
	r       *bytes.Reader
	fmap    []byte
	voc     []byte
	palette color.Palette
	Sprites []Sprite
}

func NewDecoder(r io.Reader, palette color.Palette) (*Decoder, error) {
	data, err := io.ReadAll(bufio.NewReader(r))
	if err != nil {
		return nil, err
	}
	decoder := &Decoder{
		r:       bytes.NewReader(data),
		fmap:    data,
		palette: palette,
	}
	if err := decoder.decode(); err != nil {
		return nil, err
	}
	return decoder, nil
}

func (decoder *Decoder) decode() error {
	if err := binary.Read(decoder.r, binary.LittleEndian, &decoder.header); err != nil {
		return err
	}

	frames := make([]uint32, decoder.header.PicturesCount)
	if err := binary.Read(decoder.r, binary.LittleEndian, &frames); err != nil {
		return err
	}

	decoder.voc = make([]byte, decoder.header.VocLength)
	if _, err := decoder.r.ReadAt(decoder.voc, int64(decoder.header.VocOffset)); err != nil {
		return err
	}

	decoder.Sprites = make([]Sprite, decoder.header.PicturesCount)
	for i := range decoder.Sprites {
		if err := decoder.decodeSprite(int64(frames[i]), &decoder.Sprites[i]); err != nil {
			return err
		}
	}
	return nil
}

func (decoder *Decoder) decodeFrame(offset int64) (*Frame, int64, error) {
	var h frameHeader
	if err := binary.Read(decoder.offsetReader(offset), binary.LittleEndian, &h); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, -1, nil
		}
		return nil, -1, err
	}

	if h.Lines != h.Ly {
		return nil, -1, nil
	}

	frame := &Frame{
		offset:    offset,
		header:    h,
		lineFlags: decoder.fmap[offset+int64(frameHeaderSize):],
	}

	var err error
	switch frame.Type() {
	case StandardFrame:
		err = decoder.decodeStandardFrame(frame)
	}

	return frame, int64(h.Next), err
}

func (decoder *Decoder) decodeStandardFrame(frame *Frame) error {
	coff := int64(frame.header.CData & 0x3FFF)
	if (frame.header.Options & 64) != 0 {
		coff += 16384
	}
	if (frame.header.Options & 128) != 0 {
		coff += 32768
	}

	clen := int64(frame.header.CData) >> 14
	if (frame.header.Options & 63) == 43 {
		clen += 262144
	}
	if (frame.header.Options & 63) == 42 {
		clen += 262144 * 2
	}
	shaper := decoder.offsetReader(frame.offset + int64(frameHeaderSize))
	painter := lzstd.NewDecoder(decoder.offsetReader(frame.offset+coff), decoder.voc, clen)

	err := frame.renderStd(shaper, painter, decoder.palette)
	if err == io.EOF {
		return nil
	}
	return err
}

func (decoder *Decoder) offsetReader(offset int64) *bytes.Reader {
	end := int64(len(decoder.fmap))
	if offset < 0 || offset >= end {
		return bytes.NewReader(nil)
	}
	//length := end - offset
	return bytes.NewReader(decoder.fmap[offset:])
}

func (decoder *Decoder) decodeSprite(offset int64, sprite *Sprite) error {
	for {
		frame, next, err := decoder.decodeFrame(offset)
		if err != nil {
			return err
		}
		if frame == nil {
			break
		}
		sprite.addFrame(frame)
		if next == -1 {
			break
		}
		offset += next
	}
	return nil
}
