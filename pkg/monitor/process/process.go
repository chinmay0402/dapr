package process

import (
	"strings"

	"github.com/dapr/kit/logger"
)

var log = logger.NewLogger("dapr.monitor")

func ProcessLogs(logs string) {
	// search for keywords?
	// how to create a logic flow?

	if strings.Contains(logs, "x509") {
		log.Infof("Invalid certificate, consider renewing")
	}
}
