package gsc

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"gitgub.com/cam-per/gossacks/utils"
	"golang.org/x/text/encoding/charmap"
)

const (
	key byte = 0x78
)

type header struct {
	Hash     [4]byte
	Name     [64]byte
	Offset   uint32
	Size     uint32
	Reserved uint32
	Flags    uint8
}

type Entry interface {
	fs.DirEntry
	fs.File
	fs.FileInfo
	fs.ReadDirFile
	Hash() string
	Path() string
}

type entry struct {
	path    string
	name    string
	isDir   bool
	header  *header
	entries []fs.DirEntry
	m       map[string]*entry
	ep      int
}

func newDirEntry(path string, name string) *entry {
	return &entry{
		path:   path,
		name:   name,
		header: nil,
		isDir:  true,
		m:      make(map[string]*entry),
	}
}

func newFileEntry(path string, name string, h *header) *entry {
	return &entry{
		path:   path,
		name:   name,
		header: h,
		isDir:  false,
		m:      make(map[string]*entry),
	}
}

func (e *entry) Name() string               { return e.name }
func (e *entry) IsDir() bool                { return e.isDir }
func (e *entry) Info() (fs.FileInfo, error) { return e, nil }
func (e *entry) Stat() (fs.FileInfo, error) { return e, nil }
func (e *entry) Path() string               { return e.path }
func (e *entry) ModTime() time.Time         { return time.Time{} }
func (e *entry) Sys() any                   { return nil }

func (e *entry) Type() fs.FileMode {
	if e.isDir {
		return os.ModeDir
	} else {
		return os.ModePerm
	}
}

func (e *entry) Size() int64 {
	if e.header == nil {
		return 0
	}
	return int64(e.header.Size)
}

func (e *entry) Mode() fs.FileMode {
	if e.isDir {
		return os.ModeDir
	} else {
		return os.ModePerm
	}
}

func (e *entry) ReadDir(n int) ([]fs.DirEntry, error) {
	if len(e.entries) == 0 {
		return nil, nil
	}
	start := e.ep
	if n <= 0 {
		n = len(e.entries)
		e.ep = n
	} else {
		e.ep += n
	}
	if e.ep > len(e.entries) {
		e.ep = len(e.entries)
	}
	return e.entries[start:e.ep], nil
}

func (e *entry) Hash() string {
	if e.header == nil {
		return ""
	}
	return hex.EncodeToString(e.header.Hash[:])
}

func (e *entry) exists(name string) bool {
	_, ok := e.m[name]
	return ok
}

func (e *entry) makeDir(name string) *entry {
	if v, ok := e.m[name]; ok {
		return v
	}
	v := newDirEntry(path.Join(e.path, name), name)
	e.add(v)
	return v
}

func (e *entry) add(item *entry) {
	if e.exists(item.Name()) {
		return
	}
	e.entries = append(e.entries, item)
	e.m[item.Name()] = item
}

type ArchiveReader interface {
	io.Reader
	io.ReaderAt
}

type Container struct {
	header struct {
		Descriptor [6]byte
		Version    uint16
		Key        uint16
		Entries    uint32
	}
	r          ArchiveReader
	fat        []header
	fm         map[string]*entry
	root       *entry
	dataOffset int64
}

func NewContainer(r ArchiveReader) (*Container, error) {
	container := &Container{
		r:    r,
		root: newDirEntry("/", ""),
	}
	if err := container.readHeader(); err != nil {
		return nil, err
	}
	if err := container.readFAT(); err != nil {
		return nil, err
	}
	return container, nil
}

func (container *Container) Name() string                         { return container.root.Name() }
func (container *Container) IsDir() bool                          { return true }
func (container *Container) Type() fs.FileMode                    { return os.ModeDir }
func (container *Container) Info() (fs.FileInfo, error)           { return container.root.Info() }
func (container *Container) Stat() (fs.FileInfo, error)           { return container.root.Stat() }
func (container *Container) ReadDir(n int) ([]fs.DirEntry, error) { return container.root.ReadDir(n) }

type openedFile struct {
	*entry
	sr *io.SectionReader
}

func (f *openedFile) Read(p []byte) (int, error) {
	n, err := f.sr.Read(p)
	if f.header.Flags > 0 {
		for i := 0; i < n; i++ {
			p[i] = ^p[i] ^ (^key)
		}
	}
	return n, err
}

func (f *openedFile) Close() error               { return nil }
func (f *openedFile) Stat() (fs.FileInfo, error) { return f.entry, nil }

func (container *Container) Open(name string) (fs.File, error) {
	file, ok := container.fm[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	if file.IsDir() {
		return nil, os.ErrInvalid
	}
	sr := io.NewSectionReader(container.r, container.dataOffset+int64(^file.header.Offset), int64(file.header.Size))
	return &openedFile{entry: file, sr: sr}, nil
}

func (container *Container) readHeader() error {
	return binary.Read(container.r, binary.LittleEndian, &container.header)
}

func (container *Container) readFAT() error {
	container.fat = make([]header, container.header.Entries)
	if err := binary.Read(container.r, binary.LittleEndian, &container.fat); err != nil {
		return err
	}
	container.dataOffset = int64(binary.Size(container.header) + binary.Size(container.fat))

	container.fm = map[string]*entry{
		"/": container.root,
	}
	for i, v := range container.fat {
		a := utils.CString(v.Name[:]).Decode(charmap.CodePage866)
		a = "/" + strings.ReplaceAll(a, "\\", "/")
		name := path.Base(a)
		container.createFile(a, newFileEntry(a, name, &container.fat[i]))
	}
	return nil
}

func (container *Container) createFile(path string, e *entry) {
	container.fm[path] = e
	parts := strings.Split(path, "/")
	pwd := container.root
	for i, part := range parts {
		if i == 0 {
			continue
		}
		if i == len(parts)-1 {
			pwd.add(e)
			break
		}
		pwd = pwd.makeDir(part)
		container.fm[pwd.Path()] = pwd
	}
}
