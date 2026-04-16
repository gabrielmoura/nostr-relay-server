package engine

import (
	"encoding/hex"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/pkg/negentropyV2/model"
)

func TestReconcileRoundTrip(t *testing.T) {
	clientRefs := []model.EventRef{
		mkRef(t, 100, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		mkRef(t, 101, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		mkRef(t, 102, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
	}

	serverRefs := []model.EventRef{
		mkRef(t, 100, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		mkRef(t, 101, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		mkRef(t, 105, "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"),
	}

	client, err := NewReconciler(clientRefs, Options{})
	if err != nil {
		t.Fatalf("new client reconciler: %v", err)
	}

	server, err := NewReconciler(serverRefs, Options{})
	if err != nil {
		t.Fatalf("new server reconciler: %v", err)
	}

	msg, err := client.Initiate()
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}

	have := make([]model.EventID, 0)
	need := make([]model.EventID, 0)

	for i := 0; i < 10; i++ {
		serverMsg, srvErr := server.ReconcileAsResponder(msg)
		if srvErr != nil {
			t.Fatalf("server reconcile round %d: %v", i, srvErr)
		}

		next, diff, done, cliErr := client.ReconcileAsInitiator(serverMsg)
		if cliErr != nil {
			t.Fatalf("client reconcile round %d: %v", i, cliErr)
		}

		have = append(have, diff.Have...)
		need = append(need, diff.Need...)

		if done {
			break
		}

		msg = next
	}

	if len(have) != 1 {
		t.Fatalf("unexpected have size: %d", len(have))
	}

	if len(need) != 1 {
		t.Fatalf("unexpected need size: %d", len(need))
	}

	if got, want := have[0].Hex(), "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"; got != want {
		t.Fatalf("have mismatch, got %s want %s", got, want)
	}

	if got, want := need[0].Hex(), "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"; got != want {
		t.Fatalf("need mismatch, got %s want %s", got, want)
	}
}

func mkRef(t *testing.T, ts uint64, idHex string) model.EventRef {
	t.Helper()

	buf, err := hex.DecodeString(idHex)
	if err != nil {
		t.Fatalf("decode id %s: %v", idHex, err)
	}

	var id model.EventID
	copy(id[:], buf)

	return model.EventRef{CreatedAt: ts, ID: id}
}
