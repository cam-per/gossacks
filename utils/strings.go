package utils

import (
	"bytes"

	"golang.org/x/text/encoding/charmap"
)

type CString []byte

func (c CString) NullTerminateBytes() []byte {
	i := bytes.IndexByte(c, 0)
	if i == -1 {
		return c
	} else if i == 0 {
		return nil
	} else {
		return c[:i]
	}
}

func (c CString) String() string { return string(c.NullTerminateBytes()) }

func (c CString) Decode(encoding *charmap.Charmap) string {
	buf, err := encoding.NewDecoder().Bytes(c.NullTerminateBytes())
	if err != nil {
		return c.String()
	}
	return string(buf)
}
