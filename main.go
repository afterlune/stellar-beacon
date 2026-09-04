package main

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/cmd"
	"log/slog"
	"os"
)

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @title Benetnasch 数字空间 API
// @version 1.0
// @description 面向数字空间公开内容、管理控制面和 Agent 能力的 HTTP API。Companion 机器协议另见 docs/space-companion-protocol.md。

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("command failed", "error_code", apperrors.SafeCode(err))
		os.Exit(1)
	}
}
