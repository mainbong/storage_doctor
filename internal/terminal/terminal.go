package terminal

import (
	"os"
)

// HasTTY reports whether both stdin and stdout are connected to a terminal.
func HasTTY() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
