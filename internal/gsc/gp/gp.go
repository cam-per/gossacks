package gp

import (
	"encoding/binary"
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

func (fh *frameHeader) frameType() FrameType { return FrameType(fh.Options & 0b111111) }

type FrameType uint8

const (
	StandardFrame      FrameType = 0
	NationalMaskFrame  FrameType = 1
	Transparent50Frame FrameType = 3
	Transparent75Frame FrameType = 4
	ShadowFrame        FrameType = 5
	InvalidFrame       FrameType = 0xff
)

type ImageType uint8

const (
	ImageInvalid   ImageType = 0
	ImageGP        ImageType = 1
	ImageRLC       ImageType = 2
	ImageShadowRLC ImageType = 3
)

type Frame struct {
	Type           FrameType
	CData          int64
	Dx, Dy, Lx, Ly int
}

type Sprite struct {
	frames []*Frame
}

var (
	frameHeaderSize = int64(binary.Size(frameHeader{}))
)
