package embedded

import "os"

func LoadScript(scriptPath string) (string, error) {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
