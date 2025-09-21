package gp

import (
	"encoding/binary"
	"image"
	"image/draw"
)

type header struct {
	Sign          [4]byte
	PicturesCount int16
	Reserved      int16
	VocOffset     uint32
	VocLength     uint16
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
	image.Image
	header    frameHeader
	offset    int64
	lineFlags []byte
}

func (frame *Frame) Type() FrameType { return FrameType(frame.header.Options & 0b111111) }
func (frame *Frame) Size() int       { return int(frame.header.Lx * frame.header.Ly) }

type Sprite struct {
	Frames []*Frame
	rect   image.Rectangle
}

func (sprite *Sprite) Canvas() draw.Image    { return image.NewRGBA(sprite.rect) }
func (sprite *Sprite) Rect() image.Rectangle { return sprite.rect }

func (frame *Frame) Rect() image.Rectangle {
	return image.Rect(
		int(frame.header.Dx),
		int(frame.header.Dy),
		int(frame.header.Dx+frame.header.Lx),
		int(frame.header.Dx+frame.header.Ly),
	)
}

func (sprite *Sprite) addFrame(frame *Frame) {
	if x := int(frame.header.Dx + frame.header.Lx); sprite.rect.Max.X < x {
		sprite.rect.Max.X = x
	}
	if y := int(frame.header.Dy + frame.header.Ly); sprite.rect.Max.Y < y {
		sprite.rect.Max.Y = y
	}
	sprite.Frames = append(sprite.Frames, frame)
}

var (
	frameHeaderSize = binary.Size(frameHeader{})
)
