package event

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	infradb "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/ingestion"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	internaldb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/gabrielmoura/nostr-relay-server/internal/policies"
	"github.com/gofiber/contrib/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
)

func TestDiscoveryPostgresSubscription(t *testing.T) {
	url := os.Getenv("NR_API_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("NR_API_TEST_DATABASE_URL is not set")
	}
	poolCfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if poolCfg.ConnConfig.Database != "nrserver_api_isolated_test" || (poolCfg.ConnConfig.Host != "127.0.0.1" && poolCfg.ConnConfig.Host != "localhost") {
		t.Fatal("requires dedicated loopback test database")
	}
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error { return internaldb.PrepareConn(ctx, conn) }
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	userKey, followKey, relayKey := nostr.GeneratePrivateKey(), nostr.GeneratePrivateKey(), nostr.GeneratePrivateKey()
	user, _ := nostr.GetPublicKey(userKey)
	follow, _ := nostr.GetPublicKey(followKey)
	relay, _ := nostr.GetPublicKey(relayKey)
	defer func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM event WHERE pubkey=ANY($1::text[])", []string{user, follow, relay})
	}()
	previousConfig, previousQueries, previousPolicy, previousLog := config.Cfg, internaldb.DbQueries, policies.P, log.Logger
	defer func() {
		config.Cfg, internaldb.DbQueries, policies.P, log.Logger = previousConfig, previousQueries, previousPolicy, previousLog
	}()
	config.Cfg = &config.Config{Relay: config.RelayConfig{QueryLimit: 100, QueryIDsLimit: 100, QueryAuthorsLimit: 100, QueryKindsLimit: 100, QueryTagsLimit: 100, MaxEventSize: 100000, MaxTagValueLength: 1000, FilterLimit: 100}}
	config.Cfg.RelayInformation.PrivKey, config.Cfg.RelayInformation.PubKey = relayKey, relay
	internaldb.DbQueries = infradb.New(pool)
	log.Logger = zap.NewNop()
	policies.Init()
	follows := &nostr.Event{CreatedAt: nostr.Now(), Kind: 3, Tags: nostr.Tags{{"p", follow}}}
	note := &nostr.Event{CreatedAt: nostr.Now(), Kind: 1, Tags: nostr.Tags{}, Content: "visible test note"}
	if err := follows.Sign(userKey); err != nil {
		t.Fatal(err)
	}
	if err := note.Sign(followKey); err != nil {
		t.Fatal(err)
	}
	if result, err := internaldb.DbQueries.InsertEventBatch(ctx, []*nostr.Event{follows, note}); err != nil || result.Inserted != 2 {
		t.Fatalf("seed=%#v err=%v", result, err)
	}
	ingestion.Init()
	ingestion.Start(ctx)
	defer ingestion.Stop()
	subscriber := &dto.WsServer{Ctx: ctx, Conn: &websocket.Conn{}, ChanSender: make(chan any, 8)}
	listener.SetListener("discovery-test", subscriber, nostr.Filters{{Kinds: []int{6300}}})
	defer listener.RemoveListener(subscriber)
	request := &nostr.Event{CreatedAt: nostr.Now(), Kind: 5300, Tags: nostr.Tags{{"param", "max_results", "5"}}}
	if err := request.Sign(userKey); err != nil {
		t.Fatal(err)
	}
	requester := &dto.WsServer{Ctx: ctx, ChanSender: make(chan any, 4)}
	processEvent(requester, request)
	select {
	case frame := <-requester.ChanSender:
		ack, ok := frame.(nostr.OKEnvelope)
		if !ok || !ack.OK {
			t.Fatalf("request ack=%#v", frame)
		}
	case <-ctx.Done():
		t.Fatal("request acknowledgement timed out")
	}
	var responseID string
	select {
	case frame := <-subscriber.ChanSender:
		envelope, ok := frame.(nostr.EventEnvelope)
		if !ok || envelope.SubscriptionID == nil || *envelope.SubscriptionID != "discovery-test" {
			t.Fatalf("subscription frame=%#v", frame)
		}
		var tags nostr.Tags
		if err := json.Unmarshal([]byte(envelope.Event.Content), &tags); err != nil {
			t.Fatal(err)
		}
		if len(tags) != 1 || len(tags[0]) < 2 || tags[0][1] != note.ID {
			t.Fatalf("response tags=%#v", tags)
		}
		responseID = envelope.Event.ID
		var stored int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE id=ANY($1::text[])", []string{request.ID, envelope.Event.ID}).Scan(&stored); err != nil || stored != 2 {
			t.Fatalf("stored=%d err=%v", stored, err)
		}
	case <-ctx.Done():
		t.Fatal("discovery response timed out")
	}
	// Simulate a request that survived a restart while its response did not.
	if _, err := pool.Exec(ctx, "DELETE FROM event WHERE id=$1", responseID); err != nil {
		t.Fatal(err)
	}
	for {
		discoveryJobs.Lock()
		_, busy := discoveryJobs.inFlight[request.ID]
		discoveryJobs.Unlock()
		if !busy {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("discovery slot not released")
		case <-time.After(5 * time.Millisecond):
		}
	}
	processEvent(requester, request)
	select {
	case frame := <-requester.ChanSender:
		if ack, ok := frame.(nostr.OKEnvelope); !ok || !ack.OK {
			t.Fatalf("retry ack=%#v", frame)
		}
	case <-ctx.Done():
		t.Fatal("retry acknowledgement timed out")
	}
	select {
	case frame := <-subscriber.ChanSender:
		if envelope, ok := frame.(nostr.EventEnvelope); !ok || envelope.Event.Kind != 6300 {
			t.Fatalf("retry response=%#v", frame)
		}
	case <-ctx.Done():
		t.Fatal("retry response timed out")
	}
	badParams := &nostr.Event{CreatedAt: nostr.Now(), Kind: 5300, Tags: nostr.Tags{{"param", "max_results", "0"}}}
	if err := badParams.Sign(userKey); err != nil {
		t.Fatal(err)
	}
	processEvent(requester, badParams)
	select {
	case frame := <-requester.ChanSender:
		if ack, ok := frame.(nostr.OKEnvelope); !ok || ack.OK {
			t.Fatalf("invalid params ack=%#v", frame)
		}
	case <-ctx.Done():
		t.Fatal("invalid params acknowledgement timed out")
	}
	var badCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE id=$1", badParams.ID).Scan(&badCount); err != nil || badCount != 0 {
		t.Fatalf("invalid params stored=%d err=%v", badCount, err)
	}
	invalid := &nostr.Event{CreatedAt: nostr.Now(), Kind: 5300, Tags: nostr.Tags{}}
	if err := invalid.Sign(userKey); err != nil {
		t.Fatal(err)
	}
	invalid.Content = "changed after signing"
	processEvent(requester, invalid)
	select {
	case frame := <-requester.ChanSender:
		ack, ok := frame.(nostr.OKEnvelope)
		if !ok || ack.OK {
			t.Fatalf("invalid request acknowledgement=%#v", frame)
		}
	case <-ctx.Done():
		t.Fatal("invalid request acknowledgement timed out")
	}
	var storedInvalid int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM event WHERE id=$1", invalid.ID).Scan(&storedInvalid); err != nil || storedInvalid != 0 {
		t.Fatalf("invalid request stored=%d err=%v", storedInvalid, err)
	}
}
