package logger

import (
	"io"
	"os"
)

var fileSys StandardFileSystem

type (
	FileSystem interface {
		Close(closer io.Closer) error
		Copy(writer io.Writer, reader io.Reader) (int64, error)
		Create(name string) (*os.File, error)
		Open(name string) (*os.File, error)
		Remove(name string) error
	}

	StandardFileSystem struct{}
)

func (fs StandardFileSystem) Close(closer io.Closer) error {
	return closer.Close()
}

func (fs StandardFileSystem) Copy(writer io.Writer, reader io.Reader) (int64, error) {
	return io.Copy(writer, reader)
}

func (fs StandardFileSystem) Create(name string) (*os.File, error) {
	return os.Create(name)
}

func (fs StandardFileSystem) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (fs StandardFileSystem) Remove(name string) error {
	return os.Remove(name)
}
