package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type LogLevel string

const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

type DefaultLogger struct{}

func (l *DefaultLogger) Log(level, message string, data map[string]any) {
	ts := time.Now().Format("2006-01-02 15:04:05")

	if data != nil {
		dataJSON, _ := json.Marshal(data)
		fmt.Fprintf(os.Stdout, "[%s] %s: %s %s\n", ts, level, message, string(dataJSON))
	} else {
		fmt.Fprintf(os.Stdout, "[%s] %s: %s\n", ts, level, message)
	}
}
