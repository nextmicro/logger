package logtest

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/nextmicro/logger"
)

type Buffer struct {
	buf *bytes.Buffer
	t   *testing.T
}

func Discard(t *testing.T) {
	logger.DefaultLogger = logger.New(logger.WithWriter(io.Discard))
}

func NewCollector(t *testing.T) *Buffer {
	var buf bytes.Buffer
	logger.DefaultLogger = logger.New(logger.WithWriter(&buf))

	t.Cleanup(func() {
		logger.Sync()
	})

	return &Buffer{
		buf: &buf,
		t:   t,
	}
}

func (b *Buffer) Bytes() []byte {
	return b.buf.Bytes()
}

func (b *Buffer) Content() string {
	var m map[string]interface{}
	if err := json.Unmarshal(b.buf.Bytes(), &m); err != nil {
		return ""
	}

	content, ok := m["content"]
	if !ok {
		return ""
	}

	switch val := content.(type) {
	case string:
		return val
	default:
		// err is impossible to be not nil, unmarshaled from b.buf.Bytes()
		bs, _ := json.Marshal(content)
		return string(bs)
	}
}

func (b *Buffer) Reset() {
	b.buf.Reset()
}

func (b *Buffer) String() string {
	return b.buf.String()
}
