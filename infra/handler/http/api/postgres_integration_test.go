package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	infradb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	internaldb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/groups"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

// TestPublicAPIPostgres requires the existing migrations on a dedicated local
// database. It never truncates tables or connects without the explicit opt-in.
func TestPublicAPIPostgres(t *testing.T) {
	url := os.Getenv("NR_API_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("NR_API_TEST_DATABASE_URL is not set")
	}
	poolCfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if poolCfg.ConnConfig.Database != "nrserver_api_isolated_test" || (poolCfg.ConnConfig.Host != "127.0.0.1" && poolCfg.ConnConfig.Host != "localhost") {
		t.Fatal("integration test requires the dedicated loopback database nrserver_api_isolated_test")
	}
	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	pubkey := hex.EncodeToString(random[:])
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM event WHERE pubkey=$1", pubkey) }()
	ids := make([]string, 4)
	for i := range ids {
		if _, err := rand.Read(random[:]); err != nil {
			t.Fatal(err)
		}
		ids[i] = hex.EncodeToString(random[:])
	}
	target := strings.Repeat("a", 64)
	tags := []string{"[]", `[["-"]]`, `[["e","` + target + `"]]`, `[["q","` + target + `"]]`}
	for i, id := range ids {
		_, err := pool.Exec(ctx, "INSERT INTO event (id,pubkey,created_at,kind,tags,content,sig) VALUES ($1,$2,$3,1,$4::jsonb,$5,$6)", id, pubkey, 1700000000+i, tags[i], "public test", strings.Repeat("0", 128))
		if err != nil {
			t.Fatal(err)
		}
	}
	previousCfg, previousDB := config.Cfg, internaldb.DbQueries
	config.Cfg = &config.Config{API: config.APIConfig{Enabled: true, MaxLimit: 10, MaxScan: 20}, Relay: config.RelayConfig{QueryLimit: 2, QueryAuthorsLimit: 10, QueryIDsLimit: 10, QueryKindsLimit: 10, QueryTagsLimit: 10}}
	internaldb.DbQueries = infradb.New(pool)
	t.Cleanup(func() { config.Cfg, internaldb.DbQueries = previousCfg, previousDB })
	app := fiber.New()
	handler := New(config.Cfg)
	if _, err := handler.orderedSource(ctx, nostr.Filter{Authors: []string{pubkey}, Limit: 2}, 0, true); err != nil {
		t.Fatalf("ordered source: %v", err)
	}
	handler.Register(app.Group("/api/v1"))
	for _, tc := range []struct {
		path string
		want []string
	}{
		{"/api/v1/query?author=" + pubkey + "&limit=2&sort=asc", []string{ids[0], ids[2]}},
		{"/api/v1/ids/" + target + "/related?author=" + pubkey + "&limit=2", []string{ids[3], ids[2]}},
	} {
		response, err := app.Test(httptest.NewRequest("GET", tc.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		var result eventsResponse
		err = json.NewDecoder(response.Body).Decode(&result)
		response.Body.Close()
		if err != nil || response.StatusCode != fiber.StatusOK || len(result.Events) != len(tc.want) {
			t.Fatalf("%s status=%d result=%#v err=%v", tc.path, response.StatusCode, result, err)
		}
		for i, want := range tc.want {
			if result.Events[i].ID != want {
				t.Fatalf("%s event[%d]=%s want %s", tc.path, i, result.Events[i].ID, want)
			}
		}
	}
	countFrame := fmt.Sprintf(`["COUNT","integration",{"authors":[%q],"limit":1},{"authors":[%q]}]`, pubkey, pubkey)
	frames, status := executeNIPFE(ctx, []byte(countFrame), "")
	if status != fiber.StatusOK {
		t.Fatalf("NIP-FE count status=%d frames=%#v", status, frames)
	}
	count := frames[0].([]any)[2].(map[string]int64)["count"]
	if count != 3 {
		t.Fatalf("NIP-FE count=%d, want 3 visible unique events", count)
	}
	response, err := app.Test(httptest.NewRequest("GET", "/api/v1/"+pubkey+"/1/monthly?since=1698796800&until=1701388799", nil))
	if err != nil {
		t.Fatal(err)
	}
	var monthly struct {
		Months      []monthBucket `json:"months"`
		Total       int           `json:"total"`
		Approximate bool          `json:"approximate"`
	}
	err = json.NewDecoder(response.Body).Decode(&monthly)
	response.Body.Close()
	if err != nil || response.StatusCode != fiber.StatusOK || monthly.Total != 3 || monthly.Approximate || len(monthly.Months) != 1 || monthly.Months[0].Month != "2023-11" {
		t.Fatalf("monthly status=%d result=%#v err=%v", response.StatusCode, monthly, err)
	}

	previousGroups, previousLog := groups.M, log.Logger
	defer func() { groups.M, log.Logger = previousGroups, previousLog }()
	log.Logger = zap.NewNop()
	scope := "isolated-" + pubkey
	config.Cfg.NIP29 = config.NIP29Config{Enabled: true, RelayScope: scope, GroupCreatorRole: "admin", DefaultRoles: []config.NIP29RoleConfig{{Name: "admin"}}}
	config.Cfg.RelayInformation.PrivKey = nostr.GeneratePrivateKey()
	if err := groups.Init(internaldb.DbQueries); err != nil {
		t.Fatalf("initialize groups: %v", err)
	}
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM nip29_groups WHERE relay=$1", scope) }()
	_, err = pool.Exec(ctx, `INSERT INTO nip29_groups (relay,group_id,name,private,last_metadata_update,last_admins_update,last_members_update,last_roles_update) VALUES ($1,'private-test','private',true,now(),now(),now(),now())`, scope)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO nip29_group_members (relay,group_id,user_id,role_id) SELECT $1,'private-test',$2,role_id FROM nip29_roles WHERE name='member'`, scope, pubkey)
	if err != nil {
		t.Fatal(err)
	}
	member, err := internaldb.DbQueries.IsNIP29Member(ctx, scope, "private-test", pubkey)
	if err != nil || !member {
		t.Fatalf("membership=%v err=%v", member, err)
	}
	group, exists, err := internaldb.DbQueries.GetNIP29Group(ctx, scope, "private-test")
	if err != nil || !exists || !group.Private {
		t.Fatalf("group=%#v exists=%v err=%v", group, exists, err)
	}
	groupList, _, err := internaldb.DbQueries.ListNIP29Groups(ctx, scope, 10, 0)
	if err != nil || len(groupList) != 1 {
		t.Fatalf("group list=%#v err=%v", groupList, err)
	}
	groupPage, err := internaldb.DbQueries.ListNIP29GroupsAfter(ctx, scope, nil, "", 10)
	if err != nil || len(groupPage) != 1 {
		t.Fatalf("group page=%#v err=%v", groupPage, err)
	}
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	privateID := hex.EncodeToString(random[:])
	_, err = pool.Exec(ctx, `INSERT INTO event (id,pubkey,created_at,kind,tags,content,sig) VALUES ($1,$2,1700000004,1,'[["h","private-test"]]'::jsonb,'private',$3)`, privateID, pubkey, strings.Repeat("0", 128))
	if err != nil {
		t.Fatal(err)
	}
	response, err = app.Test(httptest.NewRequest("GET", "/api/v1/query?author="+pubkey+"&limit=10", nil))
	if err != nil {
		t.Fatal(err)
	}
	var anonymous eventsResponse
	err = json.NewDecoder(response.Body).Decode(&anonymous)
	response.Body.Close()
	if err != nil || response.StatusCode != fiber.StatusOK || anonymous.Count != 3 {
		t.Fatalf("anonymous status=%d result=%#v err=%v", response.StatusCode, anonymous, err)
	}
	for _, tc := range []struct {
		name, authed string
		count        int64
	}{{"anonymous", "", 3}, {"member", pubkey, 4}, {"other", strings.Repeat("b", 64), 3}} {
		frames, status := executeNIPFE(ctx, []byte(countFrame), tc.authed)
		if status != fiber.StatusOK {
			t.Fatalf("%s count status=%d frames=%#v", tc.name, status, frames)
		}
		got := frames[0].([]any)[2].(map[string]int64)["count"]
		if got != tc.count {
			t.Fatalf("%s count=%d want=%d", tc.name, got, tc.count)
		}
	}
	reqFrame := fmt.Sprintf(`["REQ","integration",{"authors":[%q],"limit":2}]`, pubkey)
	for _, tc := range []struct {
		name, authed string
		want         []string
	}{
		{"anonymous", "", []string{ids[3], ids[2]}},
		{"member", pubkey, []string{privateID, ids[3]}},
	} {
		frames, status := executeNIPFE(ctx, []byte(reqFrame), tc.authed)
		if status != fiber.StatusOK || len(frames) != 3 {
			t.Fatalf("%s status=%d frames=%#v", tc.name, status, frames)
		}
		for i, want := range tc.want {
			frame, ok := frames[i].(nostr.EventEnvelope)
			if !ok || frame.Event.ID != want {
				t.Fatalf("%s frame[%d]=%#v want=%s", tc.name, i, frames[i], want)
			}
		}
	}
}
