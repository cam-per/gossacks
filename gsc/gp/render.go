package gp

import (
	"image"
	"image/color"
	"io"

	"gitgub.com/cam-per/gossacks/utils"
)

func (frame *Frame) renderStd(shaper, painter io.Reader, palette color.Palette) error {
	canvas := image.NewRGBA(image.Rect(0, 0, int(frame.header.Lx), int(frame.header.Ly)))
	frame.Image = canvas

	w := canvas.Bounds().Dx()

	for Y := 0; Y < int(frame.header.Lines); Y++ {
		currentX := 0

		cmd, err := utils.ReadByte(shaper)
		if err != nil {
			return err
		}

		switch {
		case cmd == 0:
			continue
		case (cmd & 0x80) != 0:
			spaceMask := byte(0)
			if (cmd & 0x40) != 0 {
				spaceMask = 0x10
			}
			pixMask := byte(0)
			if (cmd & 0x20) != 0 {
				pixMask = 0x10
			}
			count := cmd & 0x1F

			for p := 0; p < int(count); p++ {
				pack, err := utils.ReadByte(shaper)
				if err != nil {
					return err
				}
				space := int((pack & 0x0F) | spaceMask)
				pixels := int(((pack >> 4) & 0x0F) | pixMask)

				currentX += space
				for i := 0; i < pixels; i++ {
					if currentX >= w {
						break
					}

					idx, err := utils.ReadByte(painter)
					if err != nil {
						return err
					}

					canvas.Set(currentX, Y, palette[idx])
					currentX++
				}
			}
		default:
			pairs := int(cmd)
			for pi := 0; pi < pairs; pi++ {
				space, err := utils.ReadByte(shaper)
				if err != nil {
					return err
				}
				pixels, err := utils.ReadByte(shaper)
				if err != nil {
					return err
				}

				currentX += int(space)
				for i := 0; i < int(pixels); i++ {
					if currentX >= w {
						break
					}

					idx, err := utils.ReadByte(painter)
					if err != nil {
						return err
					}

					canvas.Set(currentX, Y, palette[idx])
					currentX++
				}
			}
		}
	}
	return nil
}
