package bridge

import "log"

type logBridge struct {
	module string
}

func newLogBridge(module string) *logBridge {
	return &logBridge{module: module}
}

func (l *logBridge) Log(level, msg string, data ...map[string]any) {
	prefix := "[" + l.module + "]"
	switch level {
	case "debug":
		log.Printf("%s [DEBUG] %s %v", prefix, msg, flattenData(data))
	case "info":
		log.Printf("%s [INFO] %s %v", prefix, msg, flattenData(data))
	case "warn":
		log.Printf("%s [WARN] %s %v", prefix, msg, flattenData(data))
	case "error":
		log.Printf("%s [ERROR] %s %v", prefix, msg, flattenData(data))
	default:
		log.Printf("%s [INFO] %s %v", prefix, msg, flattenData(data))
	}
}

func flattenData(data []map[string]any) map[string]any {
	if len(data) == 0 {
		return nil
	}
	return data[0]
}
