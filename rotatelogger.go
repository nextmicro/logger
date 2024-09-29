package logger

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"
	"sync"
	"time"
)

var (
	// Placeholder is a placeholder object that can be used globally.
	Placeholder PlaceholderType

	// ErrClosedRollingFile is returned when the rolling file is closed.
	ErrClosedRollingFile = errors.New("rolling file is closed")

	ErrBuffer = errors.New("buffer exceeds the limit")
)

type (
	// AnyType can be used to hold any type.
	AnyType = any
	// PlaceholderType represents a placeholder type.
	PlaceholderType = struct{}
)

const (
	defaultDirMode       = 0o755
	defaultFileMode      = 0o600
	gzipExt              = ".gz"
	backupFileDelimiter  = "-"
	sizeRotationRule     = "size"
	hourRotationRule     = "hour"
	dayRotationRule      = "day"
	megaBytes            = 1 << 20
	logPageNumber        = 2
	logPageCacheByteSize = 4096 // 4KB
)

type (
	// A RotateLogger is a Logger that can rotate log files with given rules.
	RotateLogger struct {
		filename string
		backup   string
		fp       *os.File

		syncFlush  chan struct{}
		current    *bytes.Buffer
		fullBuffer chan *bytes.Buffer

		closed   bool
		done     chan struct{}
		rule     RotateRule
		compress bool
		// can't use threading.RoutineGroup because of cycle import
		waitGroup   sync.WaitGroup
		closeOnce   sync.Once
		currentSize int64

		mu sync.Mutex
	}
)

// NewRotateLogger returns a RotateLogger with given filename and rule, etc.
func NewRotateLogger(filename string, rule RotateRule, compress bool) (*RotateLogger, error) {
	l := &RotateLogger{
		filename:   filename,
		rule:       rule,
		compress:   compress,
		done:       make(chan struct{}),
		syncFlush:  make(chan struct{}),
		fullBuffer: make(chan *bytes.Buffer, logPageNumber+1),
		current:    getBuffer(),
	}
	if err := l.initialize(); err != nil {
		return nil, err
	}

	l.startWorker()
	return l, nil
}

// flush flushes the buffer to the file.
func (l *RotateLogger) flush() {
	readyLen := len(l.fullBuffer)
	for i := 0; i < readyLen; i++ {
		buff := <-l.fullBuffer
		l.writeBuffer(buff)
		putBuffer(buff)
	}
	if l.current != nil {
		l.writeBuffer(l.current)
		putBuffer(l.current)
	}

	l.current = nil
	if l.fp != nil {
		l.fp.Sync()
	}
}

func (l *RotateLogger) startWorker() {
	l.waitGroup.Add(1)

	go func() {
		defer l.waitGroup.Done()

		defer func() {
			l.flush()
			if l.fp != nil {
				l.fp.Close()
				l.fp = nil
			}
		}()

		t := time.NewTicker(time.Millisecond * 500)
		defer t.Stop()
		for {
			select {
			case <-l.syncFlush:
				l.mu.Lock()
				l.flush()
				l.mu.Unlock()
				l.syncFlush <- struct{}{}
			case buff := <-l.fullBuffer:
				l.writeBuffer(buff)
				putBuffer(buff)
			case <-t.C:
				l.mu.Lock()
				if len(l.fullBuffer) != 0 {
					l.mu.Unlock()
					continue
				}
				// 清空buffer
				buff := l.current
				if buff == nil {
					l.mu.Unlock()
					continue
				}
				l.current = nil
				l.mu.Unlock()

				l.writeBuffer(buff)
				putBuffer(buff)
			case <-l.done:
				return
			}
		}
	}()
}

func (l *RotateLogger) Write(b []byte) (n int, err error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return 0, ErrClosedRollingFile
	}

	// check if logger is closed
	select {
	case <-l.done:
		l.mu.Unlock()
		return 0, fmt.Errorf("logger is closed")
	default:
	}

	// write to buffer
	if l.current == nil {
		l.current = getBuffer()
		if l.current == nil {
			l.mu.Unlock()
			return 0, ErrBuffer
		}
	}

	// write to buffer
	n, err = l.current.Write(b)
	if l.current.Len() > logPageCacheByteSize {
		buf := l.current
		l.current = nil
		l.mu.Unlock()
		l.fullBuffer <- buf
	} else {
		l.mu.Unlock()
	}
	return
}

func (l *RotateLogger) getBackupFilename() string {
	if len(l.backup) == 0 {
		return l.rule.BackupFileName()
	}

	return l.backup
}

func (l *RotateLogger) initialize() error {
	l.backup = l.rule.BackupFileName()

	if fileInfo, err := os.Stat(l.filename); err != nil {
		basePath := path.Dir(l.filename)
		if _, err = os.Stat(basePath); err != nil {
			if err = os.MkdirAll(basePath, defaultDirMode); err != nil {
				return err
			}
		}

		if l.fp, err = os.Create(l.filename); err != nil {
			return err
		}
	} else {
		if l.fp, err = os.OpenFile(l.filename, os.O_APPEND|os.O_WRONLY, defaultFileMode); err != nil {
			return err
		}

		l.currentSize = fileInfo.Size()
	}

	return nil
}

func (l *RotateLogger) maybeCompressFile(file string) {
	if !l.compress {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf(fmt.Sprintf("%s\n%s", r, string(debug.Stack())))
		}
	}()

	if _, err := os.Stat(file); err != nil {
		// file doesn't exist or another error, ignore compression
		return
	}

	compressLogFile(file)
}

func (l *RotateLogger) maybeDeleteOutdatedFiles() {
	files := l.rule.OutdatedFiles()
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			log.Printf("failed to remove outdated file: %s", file)
		}
	}
}

func (l *RotateLogger) postRotate(file string) {
	go func() {
		// we cannot use threading.GoSafe here, because of import cycle.
		l.maybeCompressFile(file)
		l.maybeDeleteOutdatedFiles()
	}()
}

// rotate 日志轮转
func (l *RotateLogger) rotate() error {
	// close the current file
	if err := l.close(); err != nil {
		return err
	}

	_, err := os.Stat(l.filename)
	if err == nil && len(l.backup) > 0 {
		backupFilename := l.getBackupFilename()
		err = os.Rename(l.filename, backupFilename)
		if err != nil {
			return err
		}

		l.postRotate(backupFilename)
	}

	l.backup = l.rule.BackupFileName()
	l.fp, err = os.Create(l.filename)
	return err
}

func (l *RotateLogger) writeBuffer(buff *bytes.Buffer) (int64, error) {
	if l.rule.ShallRotate(l.currentSize + int64(buff.Len())) {
		if err := l.rotate(); err != nil {
			log.Println(err)
		} else {
			l.rule.MarkRotated()
			l.currentSize = 0
		}
	}
	if l.fp == nil {
		return 0, nil
	}

	size, err := buff.WriteTo(l.fp)
	if err != nil {
		return size, err
	}
	l.currentSize += size
	return size, nil
}

// close file close the file
func (l *RotateLogger) close() (err error) {
	if l.fp == nil {
		return nil
	}

	var errs []error
	if err = l.fp.Sync(); err != nil {
		errs = append(errs, err)
	}
	err = l.fp.Close()
	if err != nil {
		errs = append(errs, err)
	}

	l.fp = nil
	return errors.Join(errs...)
}

// Close closes l.
func (l *RotateLogger) Close() (err error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.mu.Unlock()

	l.closeOnce.Do(func() {
		l.closed = true
		close(l.done)
		l.waitGroup.Wait()
		err = l.close()
	})

	return err
}

func (l *RotateLogger) Sync() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return ErrClosedRollingFile
	}
	l.mu.Unlock()

	l.syncFlush <- struct{}{}
	<-l.syncFlush
	return nil
}

func compressLogFile(file string) {
	start := time.Now()
	log.Printf("compressing log file: %s", file)
	if err := gzipFile(file, fileSys); err != nil {
		log.Printf("compress error: %s", err)
	} else {
		log.Printf("compressed log file: %s, took %s", file, time.Since(start))
	}
}

func gzipFile(file string, fsys FileSystem) (err error) {
	in, err := fsys.Open(file)
	if err != nil {
		return err
	}
	defer func() {
		if e := fsys.Close(in); e != nil {
			log.Printf("failed to close file: %s, error: %v", file, e)
		}
		if err == nil {
			// only remove the original file when compression is successful
			err = fsys.Remove(file)
		}
	}()

	out, err := fsys.Create(fmt.Sprintf("%s%s", file, gzipExt))
	if err != nil {
		return err
	}
	defer func() {
		e := fsys.Close(out)
		if err == nil {
			err = e
		}
	}()

	w := gzip.NewWriter(out)
	if _, err = fsys.Copy(w, in); err != nil {
		// failed to copy, no need to close w
		return err
	}

	return fsys.Close(w)
}
