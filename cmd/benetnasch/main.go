package main

import (
	"log/slog"
	"os"
)

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @Title benetnasch
// @Version 1.0

func main() {
	if err := execute(); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
