package cache

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"sort"

	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

type normalizedFilter struct {
	IDs     []string            `json:"ids,omitempty"`
	Authors []string            `json:"authors,omitempty"`
	Kinds   []int               `json:"kinds,omitempty"`
	Tags    map[string][]string `json:"tags,omitempty"`
	Search  string              `json:"search,omitempty"`
	Since   *uint64             `json:"since,omitempty"`
	Until   *uint64             `json:"until,omitempty"`
	Limit   *int                `json:"limit,omitempty"`
}

func BuildFilterKey(f model.Filter) (string, error) {
	n := normalizedFilter{
		IDs:     cloneAndSortStrings(f.IDs),
		Authors: cloneAndSortStrings(f.Authors),
		Kinds:   cloneAndSortInts(f.Kinds),
		Tags:    cloneTags(f.Tags),
		Search:  f.Search,
		Since:   f.Since,
		Until:   f.Until,
		Limit:   f.Limit,
	}

	buf, err := json.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("build filter key: %w", err)
	}
	sum := sha256.Sum256(buf)

	return fmt.Sprintf("f:%x", sum[:]), nil
}

func cloneAndSortStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	slices.Sort(out)

	return out
}

func cloneAndSortInts(values []int) []int {
	out := make([]int, len(values))
	copy(out, values)
	slices.Sort(out)

	return out
}

func cloneTags(tags map[string][]string) map[string][]string {
	if len(tags) == 0 {
		return nil
	}

	out := make(map[string][]string, len(tags))
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := tags[key]
		vals := cloneAndSortStrings(values)
		out[key] = vals
	}

	return out
}
