package sync

import (
	"fmt"
	"strings"
	"time"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
)

type CLIOptions struct {
	Remote    string
	Direction string
	Pk        string
	Filter    string
	Timeout   int64
}

func BuildConfig(opt CLIOptions) (*ConfSync, error) {
	if err := validateRemote(opt.Remote); err != nil {
		return nil, err
	}

	pk, err := decodePublicKey(strings.TrimSpace(opt.Pk))
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	direction, err := ParseDirection(strings.ToLower(strings.TrimSpace(opt.Direction)))
	if err != nil {
		return nil, err
	}

	if opt.Timeout < 0 {
		return nil, fmt.Errorf("invalid timeout %d: must be >= 0", opt.Timeout)
	}

	filterPayload, filters, err := parseFilterOption(opt.Filter, pk)
	if err != nil {
		return nil, err
	}

	return &ConfSync{
		Remote:      opt.Remote,
		Pk:          pk,
		Direction:   direction,
		OpenFilter:  filterPayload,
		LocalFilter: filters,
		Timeout:     time.Duration(opt.Timeout) * time.Second,
	}, nil
}

func parseFilterOption(raw string, pk string) (any, []nostr.Filter, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = "{}"
	}

	var probe any
	if err := json.Unmarshal([]byte(value), &probe); err != nil {
		return nil, nil, fmt.Errorf("invalid filter JSON: %w", err)
	}

	pk = strings.TrimSpace(pk)

	switch probe.(type) {
	case map[string]any:
		var filter nostr.Filter
		if err := json.Unmarshal([]byte(value), &filter); err != nil {
			return nil, nil, fmt.Errorf("invalid filter object: %w", err)
		}
		applyPKConstraint(&filter, pk)
		return filter, []nostr.Filter{filter}, nil
	case []any:
		var filters []nostr.Filter
		if err := json.Unmarshal([]byte(value), &filters); err != nil {
			return nil, nil, fmt.Errorf("invalid filter array: %w", err)
		}
		if len(filters) == 0 {
			return nil, nil, fmt.Errorf("invalid filter array: at least one filter is required")
		}
		for i := range filters {
			applyPKConstraint(&filters[i], pk)
		}
		return filters, filters, nil
	default:
		return nil, nil, fmt.Errorf("invalid filter JSON: expected object or array")
	}
}

func applyPKConstraint(filter *nostr.Filter, pk string) {
	if pk == "" {
		return
	}

	filter.Authors = []string{pk}
}
