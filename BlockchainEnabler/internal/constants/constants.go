package constants

import (
	"os"
	"path/filepath"
)

var homeDir, _ = os.UserHomeDir()
var EnablerDir = filepath.Join(homeDir, ".enabler", "platform")
