package pal

import (
	"image/color"
	"io"
)

type Channel uint8

const (
	ChannelAlpha Channel = iota
	ChannelR
	ChannelG
	ChannelB
	ChannelGray
	ChannelRGB
	ChannelARGB
)

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (decoder *Decoder) Decode(paletteType Channel, size int) (color.Palette, error) {
	depth := 0
	switch paletteType {
	case ChannelAlpha:
		fallthrough
	case ChannelR:
		fallthrough
	case ChannelG:
		fallthrough
	case ChannelB:
		fallthrough
	case ChannelGray:
		depth = 1
	case ChannelRGB:
		depth = 3
	case ChannelARGB:
		depth = 4
	}
	pal := make([]color.Color, size)
	buf := make([]byte, depth)

	for i := 0; i < size; i++ {
		var c color.Color
		if _, err := decoder.r.Read(buf); err != nil {
			return nil, err
		}
		switch paletteType {
		case ChannelAlpha:
			c = color.RGBA{R: 0, G: 0, B: 0, A: buf[0]}
		case ChannelR:
			c = color.RGBA{R: buf[0], G: 0, B: 0, A: 255}
		case ChannelG:
			c = color.RGBA{R: 0, G: buf[0], B: 0, A: 255}
		case ChannelB:
			c = color.RGBA{R: 0, G: 0, B: buf[0], A: 255}
		case ChannelGray:
			c = color.RGBA{R: buf[0], G: buf[0], B: buf[0], A: 255}
		case ChannelRGB:
			c = color.RGBA{R: buf[0], G: buf[1], B: buf[2], A: 255}
		case ChannelARGB:
			c = color.RGBA{R: buf[1], G: buf[2], B: buf[3], A: buf[0]}
		}
		pal[i] = c
	}
	return pal, nil
}
