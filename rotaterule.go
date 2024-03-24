package logger

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// A RotateRule interface is used to define the log rotating rules.
type RotateRule interface {
	BackupFileName() string
	MarkRotated()
	OutdatedFiles() []string
	ShallRotate(size int64) bool
}

type (
	// A DailyRotateRule is a rule to daily rotate the log files.
	DailyRotateRule struct {
		rotatedTime string
		filename    string
		delimiter   string
		days        int
		gzip        bool
	}

	// HourRotateRule a rotation rule that make the log file rotated base on hour
	HourRotateRule struct {
		rotatedTime string
		filename    string
		delimiter   string
		hours       int
		gzip        bool
	}

	// SizeLimitRotateRule a rotation rule that make the log file rotated base on size
	SizeLimitRotateRule struct {
		DailyRotateRule
		maxSize    int64
		maxBackups int
	}
)

// NewHourRotateRule new a hour rotate rule
func NewHourRotateRule(filename, delimiter string, hours int, gzip bool) *HourRotateRule {
	return &HourRotateRule{
		rotatedTime: getNowHour(),
		filename:    filename,
		delimiter:   delimiter,
		hours:       hours,
		gzip:        gzip,
	}
}

// BackupFileName returns the backup filename on rotating.
func (r *HourRotateRule) BackupFileName() string {
	return fmt.Sprintf("%s%s%s", r.filename, r.delimiter, getNowHour())
}

// MarkRotated marks the rotated time of r to be the current time.
func (r *HourRotateRule) MarkRotated() {
	r.rotatedTime = getNowHour()
}

// OutdatedFiles returns the files that exceeded the keeping hours.
func (r *HourRotateRule) OutdatedFiles() []string {
	if r.hours <= 0 {
		return nil
	}

	var pattern string
	if r.gzip {
		pattern = fmt.Sprintf("%s%s*%s", r.filename, r.delimiter, ".gz")
	} else {
		pattern = fmt.Sprintf("%s%s*", r.filename, r.delimiter)
	}

	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Printf("failed to find outdated log files, error: %s\n", err)
		return nil
	}

	var buf strings.Builder
	boundary := time.Now().Add(-time.Hour * time.Duration(r.hours)).Format(hourFormat)
	buf.WriteString(r.filename)
	buf.WriteString(r.delimiter)
	buf.WriteString(boundary)
	if r.gzip {
		buf.WriteString(gzipExt)
	}
	boundaryFile := buf.String()

	var outdates []string
	for _, file := range files {
		if file < boundaryFile {
			outdates = append(outdates, file)
		}
	}

	return outdates
}

// ShallRotate checks if the file should be rotated.
func (r *HourRotateRule) ShallRotate(_ int64) bool {
	return len(r.rotatedTime) > 0 && getNowHour() != r.rotatedTime
}

// DefaultRotateRule is a default log rotating rule, currently DailyRotateRule.
func DefaultRotateRule(filename, delimiter string, days int, gzip bool) RotateRule {
	return &DailyRotateRule{
		rotatedTime: getNowDate(),
		filename:    filename,
		delimiter:   delimiter,
		days:        days,
		gzip:        gzip,
	}
}

// BackupFileName returns the backup filename on rotating.
func (r *DailyRotateRule) BackupFileName() string {
	return fmt.Sprintf("%s%s%s", r.filename, r.delimiter, getNowDate())
}

// MarkRotated marks the rotated time of r to be the current time.
func (r *DailyRotateRule) MarkRotated() {
	r.rotatedTime = getNowDate()
}

// OutdatedFiles returns the files that exceeded the keeping days.
func (r *DailyRotateRule) OutdatedFiles() []string {
	if r.days <= 0 {
		return nil
	}

	var pattern string
	if r.gzip {
		pattern = fmt.Sprintf("%s%s*%s", r.filename, r.delimiter, gzipExt)
	} else {
		pattern = fmt.Sprintf("%s%s*", r.filename, r.delimiter)
	}

	files, err := filepath.Glob(pattern)
	if err != nil {
		Errorf("failed to delete outdated log files, error: %s", err)
		return nil
	}

	var buf strings.Builder
	boundary := time.Now().Add(-time.Hour * time.Duration(hoursPerDay*r.days)).Format(dateFormat)
	buf.WriteString(r.filename)
	buf.WriteString(r.delimiter)
	buf.WriteString(boundary)
	if r.gzip {
		buf.WriteString(gzipExt)
	}
	boundaryFile := buf.String()

	var outdates []string
	for _, file := range files {
		if file < boundaryFile {
			outdates = append(outdates, file)
		}
	}

	return outdates
}

// ShallRotate checks if the file should be rotated.
func (r *DailyRotateRule) ShallRotate(_ int64) bool {
	return len(r.rotatedTime) > 0 && getNowDate() != r.rotatedTime
}

// NewSizeLimitRotateRule returns the rotation rule with size limit
func NewSizeLimitRotateRule(filename, delimiter string, days, maxSize, maxBackups int, gzip bool) RotateRule {
	return &SizeLimitRotateRule{
		DailyRotateRule: DailyRotateRule{
			rotatedTime: getNowDateInRFC3339Format(),
			filename:    filename,
			delimiter:   delimiter,
			days:        days,
			gzip:        gzip,
		},
		maxSize:    int64(maxSize) * megaBytes,
		maxBackups: maxBackups,
	}
}

func (r *SizeLimitRotateRule) BackupFileName() string {
	dir := filepath.Dir(r.filename)
	prefix, ext := r.parseFilename()
	timestamp := getNowDateInRFC3339Format()
	return filepath.Join(dir, fmt.Sprintf("%s%s%s%s", prefix, r.delimiter, timestamp, ext))
}

func (r *SizeLimitRotateRule) MarkRotated() {
	r.rotatedTime = getNowDateInRFC3339Format()
}

func (r *SizeLimitRotateRule) OutdatedFiles() []string {
	dir := filepath.Dir(r.filename)
	prefix, ext := r.parseFilename()

	var pattern string
	if r.gzip {
		pattern = fmt.Sprintf("%s%s%s%s*%s%s", dir, string(filepath.Separator),
			prefix, r.delimiter, ext, gzipExt)
	} else {
		pattern = fmt.Sprintf("%s%s%s%s*%s", dir, string(filepath.Separator),
			prefix, r.delimiter, ext)
	}

	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("failed to delete outdated log files, error: %s\n", err)
		return nil
	}

	sort.Strings(files)

	outdated := make(map[string]PlaceholderType)

	// test if too many backups
	if r.maxBackups > 0 && len(files) > r.maxBackups {
		for _, f := range files[:len(files)-r.maxBackups] {
			outdated[f] = Placeholder
		}
		files = files[len(files)-r.maxBackups:]
	}

	// test if any too old backups
	if r.days > 0 {
		boundary := time.Now().Add(-time.Hour * time.Duration(hoursPerDay*r.days)).Format(fileTimeFormat)
		boundaryFile := filepath.Join(dir, fmt.Sprintf("%s%s%s%s", prefix, r.delimiter, boundary, ext))
		if r.gzip {
			boundaryFile += gzipExt
		}
		for _, f := range files {
			if f >= boundaryFile {
				break
			}
			outdated[f] = Placeholder
		}
	}

	var result []string
	for k := range outdated {
		result = append(result, k)
	}
	return result
}

func (r *SizeLimitRotateRule) ShallRotate(size int64) bool {
	return r.maxSize > 0 && r.maxSize < size
}

func (r *SizeLimitRotateRule) parseFilename() (prefix, ext string) {
	logName := filepath.Base(r.filename)
	ext = filepath.Ext(r.filename)
	prefix = logName[:len(logName)-len(ext)]
	return
}
