package log

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	DefaultFormat = ""
	JSONFormat    = "json"
	TextFormat    = "text"
)

type Option func(*Logger) error

func WithFormat(format string) Option {
	return func(logger *Logger) error {
		switch strings.ToLower(format) {
		case DefaultFormat:
			// skip
		case JSONFormat:
			logger.Formatter = &logrus.JSONFormatter{}
		case TextFormat:
			logger.Formatter = &logrus.TextFormatter{}
		default:
			return fmt.Errorf("unknown logger format: %q", format)
		}
		return nil
	}
}

func WithLevel(logLevel string) Option {
	return func(logger *Logger) error {
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			return err
		}
		logger.SetLevel(level)
		return nil
	}
}
