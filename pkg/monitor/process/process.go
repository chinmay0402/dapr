package process

import (
	certExpiry "github.com/dapr/dapr/pkg/monitor/scenarios/cert_expiry"
	"github.com/dapr/kit/logger"
)

var log = logger.NewLogger("dapr.monitor")

type ErrorHandler interface {
	Detect(string) bool
	Remediate() 
}

func ProcessLogs(logs string) {
	switch {
	// refactor to map for error detection
	// if fatal
	// map[string] ErrorHandler
	case certExpiry.NewCertExpiry().Detect(logs):
		remediate(certExpiry.NewCertExpiry()) // TODO: rename action
	}
	// 
	
}

func remediate(eh ErrorHandler) {
	eh.Remediate()
}
