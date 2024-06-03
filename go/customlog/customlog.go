// customlog.go
package customlog

import (
	"log"
	"os"
	"time"
)

type logWriter struct {
	writer *os.File
}

func (lw *logWriter) Write(data []byte) (int, error) {
	timestamp := time.Now().Format("2006/01/02 15:04:05.000")
	return lw.writer.Write(append([]byte(timestamp+" "), data...))
}

// Init initializes the custom logger with millisecond precision timestamps.
func Init() {
	log.SetFlags(0) // Disable default flags
	log.SetOutput(&logWriter{os.Stdout})
}
