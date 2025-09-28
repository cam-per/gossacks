package rendering

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
)

//go:embed all:shaders
var __shaders__ embed.FS

type shader struct {
	Handle     uint32
	Type       uint32
	SourceCode string
}

var (
	shadersSources  map[string][]*shader
	shadersPrograms map[string]uint32
)

func LoadShaders(f fs.FS) error {
	return fs.WalkDir(f, "/", func(name string, entry fs.DirEntry, err error) error {
		return loadShaderDirectory(f, name, entry, err)
	})
}

func CompileShaders() error {
	if err := buildShaders(); err != nil {
		return err
	}
	if err := linkShaders(); err != nil {
		return err
	}

	for _, sources := range shadersSources {
		for _, shader := range sources {
			gl.DeleteShader(shader.Handle)
			shader.Handle = 0
		}
	}

	shadersSources = nil
	return nil
}

func UseProgram(program string) {
	if shadersPrograms == nil {
		return
	}
	if _, ok := shadersPrograms[program]; !ok {
		return
	}
	gl.UseProgram(shadersPrograms[program])
}

func loadProgramSources(fsys fs.FS, entry fs.DirEntry, root, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return err
	}

	program := make([]*shader, 0, len(entries))
	programName := strings.TrimPrefix(dir, root)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := loadProgramSources(fsys, entry, root, path.Join(dir, entry.Name())); err != nil {
			return err
		}
	}

	shadersSources[programName] = program
	return nil
}

func loadShaderDirectory(fsys fs.FS, dir string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if !entry.IsDir() {
		return nil
	}

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return err
	}

	tempShaders := make(map[int]*shader)
	maxSeq := -1
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || path.Ext(name) != ".glsl" {
			continue
		}

		data, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			return err
		}

		p := strings.Split(name, ".")
		if len(p) != 3 {
			return fmt.Errorf("invalid shader file name: %s", name)
		}

		var shaderType int
		switch p[1] {
		case "vertex":
			shaderType = gl.VERTEX_SHADER
		case "fragment":
			shaderType = gl.FRAGMENT_SHADER
		case "geometry":
			shaderType = gl.GEOMETRY_SHADER
		default:
			return fmt.Errorf("unknown shader type: %s", p[1])
		}

		seq, err := strconv.Atoi(p[0])
		if err != nil {
			return fmt.Errorf("invalid shader sequence number: %s", p[0])
		}
		if seq < 0 {
			return fmt.Errorf("shader sequence number must be non-negative: %d", seq)
		}

		tempShaders[seq] = &shader{
			Type:       uint32(shaderType),
			SourceCode: string(data),
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}

	if maxSeq == -1 {
		return fmt.Errorf("no shaders found in directory: %s", dir)
	}

	finalShaders := make([]*shader, maxSeq+1)
	for seq, shader := range tempShaders {
		finalShaders[seq] = shader
	}

	for i := 0; i <= maxSeq; i++ {
		if _, ok := tempShaders[i]; !ok {
			return fmt.Errorf("missing shader with sequence number: %d in directory: %s", i, dir)
		}
	}

	shadersSources[dir] = finalShaders
	return nil
}

func buildShaders() error {
	for name, sources := range shadersSources {
		for _, shader := range sources {
			shader.Handle = gl.CreateShader(shader.Type)
			if shader.Handle == 0 {
				return fmt.Errorf("failed to create shader handle for %s", name)
			}

			csources, free := gl.Strs(shader.SourceCode + "\x00")
			gl.ShaderSource(shader.Handle, 1, csources, nil)
			free()
			gl.CompileShader(shader.Handle)

			var status int32
			gl.GetShaderiv(shader.Handle, gl.COMPILE_STATUS, &status)
			if status == gl.FALSE {
				var logLength int32
				gl.GetShaderiv(shader.Handle, gl.INFO_LOG_LENGTH, &logLength)

				logBuffer := make([]byte, logLength)

				logPtr := (*uint8)(gl.Ptr(&logBuffer[0]))

				gl.GetShaderInfoLog(shader.Handle, logLength, nil, logPtr)
				logString := gl.GoStr((*uint8)(gl.Ptr(&logBuffer[0])))

				gl.DeleteShader(shader.Handle)
				return fmt.Errorf("failed to compile shader %s:\n%s", name, logString)
			}
		}
	}
	return nil
}

func linkShaders() error {
	shadersPrograms = make(map[string]uint32)
	for name, sources := range shadersSources {
		program := gl.CreateProgram()
		for _, shader := range sources {
			gl.AttachShader(program, shader.Handle)
		}
		gl.LinkProgram(program)

		var status int32
		gl.GetProgramiv(program, gl.LINK_STATUS, &status)
		if status == gl.FALSE {
			var logLength int32
			gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

			logBuffer := make([]byte, logLength)

			logPtr := (*uint8)(gl.Ptr(&logBuffer[0]))

			gl.GetProgramInfoLog(program, logLength, nil, logPtr)
			logString := gl.GoStr((*uint8)(gl.Ptr(&logBuffer[0])))

			gl.DeleteProgram(program)
			return fmt.Errorf("failed to link program %s:\n%s", name, logString)
		}
		shadersPrograms[name] = program
	}
	return nil
}
