package utils

import (
	"encoding/hex"
	"fmt"
	"io"
	"unicode"
)

func HexDump(r io.ReaderAt, offset, length int64) error {
	buf := make([]byte, length)
	if _, err := r.ReadAt(buf, offset); err != nil {
		return err
	}

	for i := 0; i < len(buf); i += 16 {
		end := i + 16
		if end > len(buf) {
			end = len(buf)
		}
		chunk := buf[i:end]

		// hex
		hexStr := hex.EncodeToString(chunk)
		for j := 0; j < len(hexStr); j += 2 {
			fmt.Printf("%s ", hexStr[j:j+2])
		}
		// padding if not full 16
		for j := len(chunk); j < 16; j++ {
			fmt.Print("   ")
		}

		// ascii
		fmt.Print(" |")
		for _, b := range chunk {
			if unicode.IsPrint(rune(b)) {
				fmt.Printf("%c", b)
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println("|")
	}

	return nil
}
