package gp

import (
	"encoding/binary"
	"errors"
	"io"
)

type Reader interface {
	io.Reader
	io.ReaderAt
}

type Decoder struct {
	r       Reader
	header  header
	sprites []Sprite
	palette []byte
}

func NewDecoder(r Reader) (*Decoder, error) {
	decoder := &Decoder{r: r}
	if err := decoder.decode(); err != nil {
		return nil, err
	}
	return decoder, nil
}

func (decoder *Decoder) decode() error {
	if err := binary.Read(decoder.r, binary.LittleEndian, &decoder.header); err != nil {
		return err
	}

	if err := decoder.decodePalette(); err != nil {
		return err
	}

	frames := make([]uint32, decoder.header.PicturesCount)
	if err := binary.Read(decoder.r, binary.LittleEndian, &frames); err != nil {
		return err
	}

	decoder.sprites = make([]Sprite, decoder.header.PicturesCount)
	for i := range decoder.sprites {
		if err := decoder.decodeSprite(int64(frames[i]), &decoder.sprites[i]); err != nil {
			return err
		}
	}
	return nil
}

func (decoder *Decoder) decodeFrame(offset int64) (*Frame, int64, error) {
	var h frameHeader
	r := io.NewSectionReader(decoder.r, offset, frameHeaderSize)
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, -1, nil
		}
		return nil, -1, err
	}

	frame := &Frame{
		Type:  FrameType(h.Options & 0b111111),
		CData: int64(h.CData),
		Dx:    int(h.Dx),
		Dy:    int(h.Dy),
		Lx:    int(h.Lx),
		Ly:    int(h.Ly),
	}

	return frame, int64(h.Next), nil
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
		sprite.frames = append(sprite.frames, frame)
		if next == -1 {
			break
		}
		offset += next
	}
	return nil
}

func (decoder *Decoder) decodePalette() error {
	decoder.palette = make([]byte, decoder.header.VocLength)
	_, err := decoder.r.ReadAt(decoder.palette, int64(decoder.header.VocOffset))
	return err
	//
	//const paletteSize = 256
	//const entrySize = 3
	//const blockSize = paletteSize * entrySize
	//decoder.palette = make(color.Palette, decoder.header.VocLength/entrySize)
	//
	//for i := 0; i < cap(decoder.palette); i++ {
	//	off := i * 3
	//	if off >= len(buf) {
	//		break
	//	}
	//	decoder.palette[i] = color.RGBA{
	//		R: buf[off+0],
	//		G: buf[off+1],
	//		B: buf[off+2],
	//		A: 255,
	//	}
	//}
	//return nil
}

//func (decoder *Decoder) decodeSprite(offset int64, sprite *Sprite) error {
//W := int(sprite.header.Lx)
//H := int(sprite.header.Lines)
//if W <= 0 || H <= 0 {
//	return fmt.Errorf("invalid dims %d x %d", W, H)
//}
//
//// Маска починається на frameFileOffset + 23
//maskOffset := offset + 23
//
//img := image.NewAlpha(image.Rect(0, 0, W, H))
//// за замовчуванням весь алфа = 0 (прозорий)
//
//// current file pointer into mask stream
//mpos := maskOffset
//
//for y := 0; y < H; y++ {
//	// прочитати керуючий байт
//	var cmdb [1]byte
//	if _, err := decoder.r.ReadAt(cmdb[:], mpos); err != nil {
//		if err == io.EOF {
//			break
//		}
//		return err
//	}
//	mpos += 1
//	cmd := cmdb[0]
//	if cmd == 0 {
//		// пустий рядок
//		continue
//	}
//
//	if (cmd & 0x80) == 0 {
//		// simple line: cmd = number of segments
//		segs := int(cmd)
//		x := 0 // початково лівий край кадра
//		for s := 0; s < segs; s++ {
//			// read space and run (2 bytes)
//			var buf [2]byte
//			if _, err := decoder.r.ReadAt(buf[:], mpos); err != nil {
//				return err
//			}
//			mpos += 2
//			space := int(buf[0])
//			run := int(buf[1])
//			x += space
//			// mark run pixels
//			for k := 0; k < run; k++ {
//				if x >= 0 && x < W {
//					img.SetAlpha(x, y, color.Alpha{A: 0xFF})
//				}
//				x++
//			}
//		}
//	} else {
//		// complex line
//		hasSpaceMask := (cmd & 0x40) != 0
//		hasPixMask := (cmd & 0x20) != 0
//		segs := int(cmd & 0x1F)
//		x := 0
//		for s := 0; s < segs; s++ {
//			var b [1]byte
//			if _, err := decoder.r.ReadAt(b[:], mpos); err != nil {
//				return err
//			}
//			mpos += 1
//			bb := b[0]
//			space := int(bb & 0x0F)
//			pix := int(bb >> 4)
//			if hasSpaceMask {
//				space |= 16
//			}
//			if hasPixMask {
//				pix |= 16
//			}
//			x += space
//			for k := 0; k < pix; k++ {
//				if x >= 0 && x < W {
//					img.SetAlpha(x, y, color.Alpha{A: 0xFF})
//				}
//				x++
//			}
//		}
//	}
//}
//
//return nil
//}

func (decoder *Decoder) Sprites() []Sprite {
	return decoder.sprites
}
