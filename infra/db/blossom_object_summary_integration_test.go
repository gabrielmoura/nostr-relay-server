package db

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestBlossomObjectSummaryMatchesFullQueryPostgres(t *testing.T) {
	dsn := os.Getenv("NR_API_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set NR_API_TEST_DATABASE_URL to the isolated local test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" && parsed.Hostname() != "::1") || parsed.Path != "/nrserver_api_isolated_test" {
		t.Fatal("integration test requires local database nrserver_api_isolated_test")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	for _, table := range []string{"objects", "blossom_objects_admin", "profiles"} {
		if _, err := tx.Exec(ctx, "CREATE TEMP TABLE "+table+" (LIKE public."+table+" INCLUDING DEFAULTS) ON COMMIT DROP"); err != nil {
			t.Fatal(err)
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO objects (hash, created_at, mime_type, size, public_key) VALUES
		('summary-a', '2024-01-02', 'image/png', 10, 'user-a'),
		('summary-b', '2024-01-01', 'image/jpeg', 20, 'user-b')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO blossom_objects_admin (hash, review_state, exif_status, thumbnail_hash) VALUES
		('summary-a', 'flagged', 'ready', 'thumb-a'), ('summary-b', 'ready', 'pending', 'thumb-b')`)
	if err != nil {
		t.Fatal(err)
	}
	queries := New(tx)
	for _, filter := range []BlossomObjectFilters{{}, {Query: "image"}, {Query: "flagged"}, {SHA256: "summary-b"}, {MIMEType: "image/png"}} {
		for _, offset := range []int{0, 1} {
			full, fullTotal, err := queries.ListBlossomObjects(ctx, filter, 1, offset)
			if err != nil {
				t.Fatal(err)
			}
			summary, summaryTotal, err := queries.ListBlossomObjectSummaries(ctx, filter, 1, offset)
			if err != nil {
				t.Fatal(err)
			}
			if fullTotal != summaryTotal || len(full) != len(summary) {
				t.Fatalf("filter=%+v offset=%d: totals/length differ", filter, offset)
			}
			for i := range full {
				a, b := full[i], summary[i]
				if a.Hash != b.Hash || a.MIMEType != b.MIMEType || a.Size != b.Size || !a.CreatedAt.Equal(b.CreatedAt) || a.ThumbnailHash != b.ThumbnailHash || a.ReviewState != b.ReviewState || a.ExifStatus != b.ExifStatus {
					t.Fatalf("filter=%+v offset=%d: full=%+v summary=%+v", filter, offset, a, b)
				}
			}
		}
	}
}
