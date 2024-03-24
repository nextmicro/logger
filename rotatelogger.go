package logger

import (
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

// Placeholder is a placeholder object that can be used globally.
var Placeholder PlaceholderType

type (
	// AnyType can be used to hold any type.
	AnyType = any
	// PlaceholderType represents a placeholder type.
	PlaceholderType = struct{}
)

const (
	dateFormat          = "2006-01-02"
	hourFormat          = "2006010215"
	fileTimeFormat      = time.RFC3339
	hoursPerDay         = 24
	bufferSize          = 100
	defaultDirMode      = 0o755
	defaultFileMode     = 0o600
	gzipExt             = ".gz"
	backupFileDelimiter = "-"
	sizeRotationRule    = "size"
	hourRotationRule    = "hour"
	megaBytes           = 1 << 20
)

// ErrLogFileClosed is an error that indicates the log file is already closed.
var ErrLogFileClosed = errors.New("error: log file closed")

type (
	// A RotateLogger is a Logger that can rotate log files with given rules.
	RotateLogger struct {
		filename string
		backup   string
		fp       *os.File
		channel  chan []byte
		done     chan PlaceholderType
		rule     RotateRule
		compress bool
		// can't use threading.RoutineGroup because of cycle import
		waitGroup   sync.WaitGroup
		closeOnce   sync.Once
		currentSize int64
	}
)

// NewRotateLogger returns a RotateLogger with given filename and rule, etc.
func NewRotateLogger(filename string, rule RotateRule, compress bool) (*RotateLogger, error) {
	l := &RotateLogger{
		filename: filename,
		channel:  make(chan []byte, bufferSize),
		done:     make(chan PlaceholderType),
		rule:     rule,
		compress: compress,
	}
	if err := l.initialize(); err != nil {
		return nil, err
	}

	l.startWorker()
	return l, nil
}

func (l *RotateLogger) Write(data []byte) (int, error) {
	select {
	case l.channel <- data:
		return len(data), nil
	case <-l.done:
		log.Println(string(data))
		return 0, ErrLogFileClosed
	}
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

func (l *RotateLogger) rotate() error {
	if l.fp != nil {
		err := l.fp.Close()
		l.fp = nil
		if err != nil {
			return err
		}
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

func (l *RotateLogger) startWorker() {
	l.waitGroup.Add(1)

	go func() {
		defer l.waitGroup.Done()

		for {
			select {
			case event := <-l.channel:
				l.write(event)
			case <-l.done:
				// avoid losing logs before closing.
				for {
					select {
					case event := <-l.channel:
						l.write(event)
					default:
						return
					}
				}
			}
		}
	}()
}

func (l *RotateLogger) write(v []byte) {
	if l.rule.ShallRotate(l.currentSize + int64(len(v))) {
		if err := l.rotate(); err != nil {
			log.Println(err)
		} else {
			l.rule.MarkRotated()
			l.currentSize = 0
		}
	}
	if l.fp != nil {
		l.fp.Write(v)
		l.currentSize += int64(len(v))
	}
}

// Close closes l.
func (l *RotateLogger) Close() (err error) {
	l.closeOnce.Do(func() {
		close(l.done)
		l.waitGroup.Wait()
		if err = l.fp.Sync(); err != nil {
			return
		}
		err = l.fp.Close()
	})

	return err
}

func (l *RotateLogger) Sync() error {
	return l.Close()
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

func getNowDate() string {
	return time.Now().Format(dateFormat)
}

func getNowHour() string {
	return time.Now().Format(hourFormat)
}

func getNowDateInRFC3339Format() string {
	return time.Now().Format(fileTimeFormat)
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
