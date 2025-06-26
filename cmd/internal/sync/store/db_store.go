package store

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"go.uber.org/zap"

	"github.com/fiatjaf/eventstore"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr"
)

var _ eventstore.Store = (*DBStore)(nil)

// NewDBStore
func NewDBStore(ctx context.Context) (store *DBStore) {
	if err := db.Init(ctx); err != nil {
		log.Logger.Fatal("Erro ao iniciar conexão com o banco de dados", zap.Error(err))
	}
	store = &DBStore{}
	return
}

type DBStore struct{}

func (s *DBStore) Init() error {
	return nil
}

func (s *DBStore) Close() {
}

// QueryEvents apenas delega para QueryEventsChan, retornando o canal original.
func (s *DBStore) QueryEvents(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error) {
	return db.DbQueries.QueryEventsChan(ctx, filter)
}
func (s *DBStore) QuerySync(ctx context.Context, filter nostr.Filter) ([]*nostr.Event, error) {
	return db.DbQueries.QueryEvents(ctx, filter)
}

func (s *DBStore) SaveEvent(ctx context.Context, evt *nostr.Event) error {
	return db.DbQueries.InsertEvent(ctx, evt)
}

func (s *DBStore) DeleteEvent(ctx context.Context, evt *nostr.Event) error {
	//return db.DbQueries.DeleteEventByID(ctx, evt.ID)
	return nil
}
func (s *DBStore) CountEvents(ctx context.Context, filter nostr.Filter) (int64, error) {
	return db.DbQueries.CountEvents(ctx, filter)
}
