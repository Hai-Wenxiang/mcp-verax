package scanner

import "os"

func osWriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
