package server

import "strings"

func handle(raw string) string {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return "-ERR empty command"
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "PING":
		return "+PONG"
	default:
		return "-ERR unknown command '" + parts[0] + "'"
	}
}
