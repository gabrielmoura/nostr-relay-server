package db

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDailyStatsSQLAppliesClosedWindowAndExcludesDeletedEvents(t *testing.T) {
	for _, query := range []string{dailyStatsSummarySQL, dailyStatsTopKindsSQL, dailyStatsTopAuthorsSQL, dailyStatsTopTagsSQL} {
		require.Contains(t, query, "created_at >= $1::bigint")
		require.Contains(t, query, "created_at < $2::bigint")
		require.Contains(t, query, "deleted_by IS NULL")
	}
}

func TestDailyStatsTopTagsSQLNormalizesAndDeduplicatesEachEventTag(t *testing.T) {
	require.Contains(t, dailyStatsTopTagsSQL, "SELECT DISTINCT event.id, LOWER(BTRIM(tag->>1)) AS tag")
	require.Contains(t, dailyStatsTopTagsSQL, "ORDER BY count DESC, tag ASC")
	require.Contains(t, dailyStatsSummarySQL, "COUNT(*) FILTER")
	require.Contains(t, dailyStatsSummarySQL, "tag->>0 = 't'")
}

func TestDailyStatsRankingLimitsAreFixed(t *testing.T) {
	require.Equal(t, dailyStatsTopKindsLimit, strings.Count(dailyStatsTopKindsSQL, "LIMIT 10;")*10)
	require.Equal(t, dailyStatsTopAuthorsLimit, strings.Count(dailyStatsTopAuthorsSQL, "LIMIT 10;")*10)
	require.Equal(t, dailyStatsTopTagsLimit, strings.Count(dailyStatsTopTagsSQL, "LIMIT 20;")*20)
}
