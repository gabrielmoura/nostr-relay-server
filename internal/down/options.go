package down

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type CLIOptions struct {
	PublicKey  string
	RelayURL   []string
	Kinds      []int
	Tags       []string
	Mentioned  bool
	Timeout    int
	Filter     string
	FilterFile string
	Merge      string
}

type DownloadOptions struct {
	RelayURLs []string
	Timeout   time.Duration
	Filter    nostr.Filter
	Merge     FilterMergeStrategy
}

type FilterMergeStrategy string

const (
	MergeOverride       FilterMergeStrategy = "override"
	MergeStrictConflict FilterMergeStrategy = "strict-conflict"
)

func BuildOptions(raw CLIOptions) (*DownloadOptions, error) {
	relays := normalizeRelayURLs(raw.RelayURL)
	if len(relays) == 0 {
		return nil, fmt.Errorf("invalid --relay-url: at least one relay URL is required")
	}

	if raw.Timeout <= 0 {
		return nil, fmt.Errorf("invalid --timeout %d: must be greater than 0", raw.Timeout)
	}

	merge, err := parseMergeStrategy(raw.Merge)
	if err != nil {
		return nil, err
	}

	pk, err := normalizePublicKey(strings.TrimSpace(raw.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("invalid --public-key: %w", err)
	}

	if raw.Mentioned && pk == "" {
		return nil, fmt.Errorf("invalid flag combination: --mentioned requires --public-key")
	}

	filterPayload, err := resolveFilterPayload(raw.Filter, raw.FilterFile)
	if err != nil {
		return nil, err
	}

	filter, err := parseFilterOption(filterPayload)
	if err != nil {
		return nil, err
	}

	if err := mergeFlagFilter(&filter, pk, raw.Mentioned, raw.Kinds, raw.Tags, merge); err != nil {
		return nil, err
	}

	return &DownloadOptions{
		RelayURLs: relays,
		Timeout:   time.Duration(raw.Timeout) * time.Second,
		Filter:    filter,
		Merge:     merge,
	}, nil
}

func parseMergeStrategy(raw string) (FilterMergeStrategy, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return MergeOverride, nil
	}

	strategy := FilterMergeStrategy(strings.ToLower(value))
	switch strategy {
	case MergeOverride, MergeStrictConflict:
		return strategy, nil
	default:
		return "", fmt.Errorf("invalid --filter-merge %q: expected one of override/strict-conflict", raw)
	}
}

func resolveFilterPayload(rawInline string, filePath string) (string, error) {
	inline := strings.TrimSpace(rawInline)
	path := strings.TrimSpace(filePath)

	if inline != "" && path != "" {
		return "", fmt.Errorf("invalid filter source: use only one of --filter or --filter-file")
	}

	if path == "" {
		return inline, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read --filter-file %q: %w", path, err)
	}

	return strings.TrimSpace(string(contents)), nil
}

func parseFilterOption(raw string) (nostr.Filter, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nostr.Filter{}, nil
	}

	var probe any
	if err := json.Unmarshal([]byte(value), &probe); err != nil {
		return nostr.Filter{}, fmt.Errorf("invalid --filter JSON: %w", err)
	}

	if _, ok := probe.(map[string]any); !ok {
		return nostr.Filter{}, fmt.Errorf("invalid --filter JSON: expected object")
	}

	var filter nostr.Filter
	if err := json.Unmarshal([]byte(value), &filter); err != nil {
		return nostr.Filter{}, fmt.Errorf("invalid --filter payload: %w", err)
	}

	return filter, nil
}

func mergeFlagFilter(
	filter *nostr.Filter,
	publicKey string,
	mentioned bool,
	kinds []int,
	tags []string,
	merge FilterMergeStrategy,
) error {
	if merge == MergeStrictConflict {
		if err := validateMergeConflicts(filter, publicKey, mentioned, kinds, tags); err != nil {
			return err
		}
	}

	if len(kinds) > 0 {
		filter.Kinds = append([]int(nil), kinds...)
	}

	if trimmedTags := normalizeTags(tags); len(trimmedTags) > 0 {
		if filter.Tags == nil {
			filter.Tags = nostr.TagMap{}
		}
		filter.Tags["t"] = trimmedTags
	}

	if publicKey == "" {
		return nil
	}

	if mentioned {
		if filter.Tags == nil {
			filter.Tags = nostr.TagMap{}
		}
		filter.Tags["p"] = []string{publicKey}
		filter.Authors = nil
		return nil
	}

	filter.Authors = []string{publicKey}
	return nil
}

func validateMergeConflicts(
	filter *nostr.Filter,
	publicKey string,
	mentioned bool,
	kinds []int,
	tags []string,
) error {
	if len(kinds) > 0 && len(filter.Kinds) > 0 && !equalIntSets(filter.Kinds, kinds) {
		return fmt.Errorf("filter conflict on kinds: JSON filter and --kinds differ")
	}

	normalizedTags := normalizeTags(tags)
	if len(normalizedTags) > 0 {
		existing := filter.Tags["t"]
		if len(existing) > 0 && !equalStringSets(existing, normalizedTags) {
			return fmt.Errorf("filter conflict on #t: JSON filter and --tags differ")
		}
	}

	if publicKey == "" {
		if mentioned {
			return fmt.Errorf("invalid flag combination: --mentioned requires --public-key")
		}
		return nil
	}

	if mentioned {
		existingP := filter.Tags["p"]
		if len(existingP) > 0 && !equalStringSets(existingP, []string{publicKey}) {
			return fmt.Errorf("filter conflict on #p: JSON filter and --mentioned/--public-key differ")
		}

		if len(filter.Authors) > 0 {
			return fmt.Errorf("filter conflict: JSON filter has authors but --mentioned requires #p filtering")
		}

		return nil
	}

	if len(filter.Authors) > 0 && !equalStringSets(filter.Authors, []string{publicKey}) {
		return fmt.Errorf("filter conflict on authors: JSON filter and --public-key differ")
	}

	return nil
}

func equalIntSets(left, right []int) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}

	leftSet := make(map[int]struct{}, len(left))
	for _, item := range left {
		leftSet[item] = struct{}{}
	}

	rightSet := make(map[int]struct{}, len(right))
	for _, item := range right {
		rightSet[item] = struct{}{}
	}

	if len(leftSet) != len(rightSet) {
		return false
	}

	for key := range leftSet {
		if _, ok := rightSet[key]; !ok {
			return false
		}
	}

	return true
}

func equalStringSets(left, right []string) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}

	leftNorm := normalizeStringSet(left)
	rightNorm := normalizeStringSet(right)

	if len(leftNorm) != len(rightNorm) {
		return false
	}

	for i := range leftNorm {
		if leftNorm[i] != rightNorm[i] {
			return false
		}
	}

	return true
}

func normalizeStringSet(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean == "" {
			continue
		}
		set[clean] = struct{}{}
	}

	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)

	return result
}

func normalizePublicKey(pk string) (string, error) {
	if pk == "" {
		return "", nil
	}

	if strings.HasPrefix(pk, "npub") {
		_, raw, err := nip19.Decode(pk)
		if err != nil {
			return "", err
		}

		decoded, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("invalid npub payload")
		}

		return decoded, nil
	}

	return pk, nil
}

func normalizeRelayURLs(relays []string) []string {
	if len(relays) == 0 {
		return nil
	}

	clean := make([]string, 0, len(relays))
	seen := make(map[string]struct{}, len(relays))
	for _, relay := range relays {
		value := strings.TrimSpace(relay)
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		clean = append(clean, value)
	}

	return clean
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	clean := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		clean = append(clean, value)
	}

	return clean
}
