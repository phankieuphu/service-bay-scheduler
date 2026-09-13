package config

import (
	"vehicle-service/pkg/logger"
)

// ToOptions converts the Logger config into pkg/logger.Options for logger.Init.
func (l Logger) ToOptions() logger.Options {
	format := logger.FormatJSON
	if logger.Format(l.Format) == logger.FormatText {
		format = logger.FormatText
	}

	return logger.Options{
		Level:     l.Level,
		Format:    format,
		AddSource: l.AddSource,
	}
}
