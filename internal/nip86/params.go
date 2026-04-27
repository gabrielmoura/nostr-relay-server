package nip86

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
)

func parseTargetAndReason(params []json.RawMessage) (string, string, error) {
	target, err := parseSingleString(params, 0, "target")
	if err != nil {
		return "", "", err
	}
	reason := ""
	if len(params) > 1 {
		reason, err = parseSingleString(params, 1, "reason")
		if err != nil {
			return "", "", err
		}
	}
	return target, reason, nil
}

func parseSingleString(params []json.RawMessage, index int, field string) (string, error) {
	if len(params) <= index {
		return "", fmt.Errorf("missing %s parameter", field)
	}
	var value string
	if err := json.Unmarshal(params[index], &value); err != nil {
		return "", fmt.Errorf("invalid %s parameter", field)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty %s parameter", field)
	}
	return value, nil
}

func normalizeIP(ip string) (string, error) {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return "", fmt.Errorf("invalid ip parameter")
	}
	return addr.String(), nil
}

func normalizeHex32(value string, field string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 64 {
		return "", fmt.Errorf("invalid %s parameter", field)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", fmt.Errorf("invalid %s parameter", field)
	}
	return value, nil
}
