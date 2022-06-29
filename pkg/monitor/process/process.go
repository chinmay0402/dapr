package process

import (
	"strings"
	certExpiry "github.com/dapr/dapr/pkg/monitor/scenarios/cert_expiry"
	"github.com/dapr/kit/logger"
)

var log = logger.NewLogger("dapr.monitor")

type ErrorHandler interface {
	Detect(string) bool
	Remediate() 
}

func ProcessLogs(logs string) {
	if(strings.Contains(logs, "fatal")){
		switch {
		case certExpiry.NewCertExpiry().Detect(logs):
			remediate(certExpiry.NewCertExpiry()) 
		}
	}
	
}

func remediate(eh ErrorHandler) {
	eh.Remediate()
}
