package logger

import (
	"log"
	"log/slog"
	"os"
)

// Log is the global structured JSON logger
var Log *slog.Logger

// Init sets up structured JSON logging on stdout and redirects legacy log.Printf
// calls to JSON format automatically — no need to change existing code.
func Init() {
	Log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(Log)

	// Redirect standard library log.Printf → slog JSON
	log.SetFlags(0)
	log.SetOutput(&slogWriter{logger: Log})

	slog.Info("structured JSON logging initialized")
}

// slogWriter adapts slog.Logger to io.Writer for log.SetOutput
type slogWriter struct {
	logger *slog.Logger
}

func (sw *slogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	sw.logger.Info(msg)
	return len(p), nil
}
