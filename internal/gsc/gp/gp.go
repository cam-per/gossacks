package gp

import (
	"encoding/binary"
	"image"
)

type header struct {
	Sign          [4]byte
	PicturesCount int16
	Reserved      int16
	VocOffset     int32
	VocLength     int16
}

type frameHeader struct {
	Next    int32
	Dx, Dy  int16
	Lx, Ly  int16
	Pack    uint32
	Options uint8
	CData   uint32
	Lines   int16
}

type Frame struct {
	header frameHeader
	image.Paletted
}

type Texture struct{}

type Sprite struct {
	images image.Paletted
}

var (
	headerSize = uint32(binary.Size(header{}))
	frameSize  = uint32(binary.Size(frameHeader{}))
)
