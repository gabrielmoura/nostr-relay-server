package relaysync

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestMemoryJournalReplayAndRetention(t *testing.T) {
	provider := NewMemoryJournalProvider(7, 2)
	provider.Append(JournalChange{Type: ChangeUpsert})
	provider.Append(JournalChange{Type: ChangeUpsert})
	last := provider.Append(JournalChange{Type: ChangeDelete})
	if _, err := provider.OpenJournal(context.Background(), Scope{}, Cursor{Epoch: 7, Sequence: 0}); !errors.Is(err, ErrJournalGap) {
		t.Fatalf("expected gap, got %v", err)
	}
	journal, err := provider.OpenJournal(context.Background(), Scope{}, Cursor{Epoch: 7, Sequence: 2})
	if err != nil {
		t.Fatal(err)
	}
	change, err := journal.Next(context.Background())
	if err != nil || change.Cursor != last {
		t.Fatalf("unexpected replay: %+v %v", change, err)
	}
	if _, err := journal.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
	_ = journal.Close()
}

func TestMemoryJournalRejectsEpochChange(t *testing.T) {
	provider := NewMemoryJournalProvider(3, 2)
	if _, err := provider.OpenJournal(context.Background(), Scope{}, Cursor{Epoch: 4}); !errors.Is(err, ErrJournalEpochChanged) {
		t.Fatalf("got %v", err)
	}
}
