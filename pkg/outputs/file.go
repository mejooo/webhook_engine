package outputs

import (
	"context"
	"encoding/json"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

type FileDriver struct {
	log  *logrus.Logger
	w    *lumberjack.Logger
	path string

	maxSizeMB  int
	maxBackups int
	maxAgeDays int
	compress   bool
}

func NewFile(log *logrus.Logger, path string, maxSizeMB, maxBackups, maxAgeDays int, compress bool) *FileDriver {
	return &FileDriver{
		log:        log,
		path:       path,
		maxSizeMB:  maxSizeMB,
		maxBackups: maxBackups,
		maxAgeDays: maxAgeDays,
		compress:   compress,
	}
}

func (d *FileDriver) Name() string { return "file" }

func (d *FileDriver) Start(ctx context.Context) error {
	d.w = &lumberjack.Logger{
		Filename:   d.path,
		MaxSize:    d.maxSizeMB,
		MaxBackups: d.maxBackups,
		MaxAge:     d.maxAgeDays,
		Compress:   d.compress,
	}
	return nil
}
func (d *FileDriver) Stop(ctx context.Context) error {
	if d.w != nil {
		return d.w.Close()
	}
	return nil
}

func (d *FileDriver) Write(ctx context.Context, batch []Record) error {
	for _, r := range batch {
		// NDJSON line: {"type":"zoom","body":<raw zoom json>}
		line, _ := json.Marshal(struct {
			Type string          `json:"type"`
			Body json.RawMessage `json:"body"`
		}{"zoom", json.RawMessage(r.Body)})
		if _, err := d.w.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return nil
}
