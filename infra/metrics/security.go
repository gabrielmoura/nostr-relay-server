package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	NostrSecurityMessageRejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_security_message_rejected_total",
			Help: "Total incoming websocket messages rejected by security reason.",
		},
		[]string{"reason"},
	)
	NostrSecurityEventRejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_security_event_rejected_total",
			Help: "Total EVENT rejections by security reason.",
		},
		[]string{"reason"},
	)
	NostrSecurityReqRejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_security_req_rejected_total",
			Help: "Total REQ or COUNT rejections by security reason.",
		},
		[]string{"reason"},
	)
	NostrSecurityConnectionsRejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_security_connections_rejected_total",
			Help: "Total websocket connections rejected by security reason.",
		},
		[]string{"reason"},
	)
	NostrSecurityDefenseActionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_security_defense_actions_total",
			Help: "Total progressive defense actions by scope and action.",
		},
		[]string{"scope", "action"},
	)
	NostrSecurityWhitelistBypassTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_security_whitelist_bypass_total",
			Help: "Total whitelist bypasses by subject and flow.",
		},
		[]string{"subject", "flow"},
	)
	NostrSecuritySignatureChecksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nostr_security_signature_checks_total",
			Help: "Total Schnorr signature verification results by status.",
		},
		[]string{"result"},
	)
)

func RegisterSecurityMetrics() {
	prometheus.MustRegister(
		NostrSecurityMessageRejectedTotal,
		NostrSecurityEventRejectedTotal,
		NostrSecurityReqRejectedTotal,
		NostrSecurityConnectionsRejectedTotal,
		NostrSecurityDefenseActionsTotal,
		NostrSecurityWhitelistBypassTotal,
		NostrSecuritySignatureChecksTotal,
	)
}
