package privacy

import (
	"context"
	"reflect"
	"testing"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"go.uber.org/zap"
)

type authTestService struct {
	closed bool
	urls   []string
}

func (s *authTestService) Start(context.Context, int) error { return nil }
func (s *authTestService) Addresses() []string              { return nil }
func (s *authTestService) Close() error {
	s.closed = true
	return nil
}
func (s *authTestService) Name() string           { return "test" }
func (s *authTestService) Status() StatusSnapshot { return StatusSnapshot{} }
func (s *authTestService) AuthURLs() []string     { return append([]string(nil), s.urls...) }

func TestManagerAuthURLsAndCloseClearsActiveRegistry(t *testing.T) {
	service := &authTestService{urls: []string{"ws://publishedonion.onion"}}
	manager := NewManager(config.PrivacyConfig{}, zap.NewNop())
	manager.services = []Service{service}
	setActiveAuthURLs(service.urls)
	t.Cleanup(func() { setActiveAuthURLs(nil) })

	if got := manager.AuthURLs(); !reflect.DeepEqual(got, service.urls) {
		t.Fatalf("AuthURLs() = %#v, want %#v", got, service.urls)
	}
	manager.Close()
	if !service.closed {
		t.Fatal("Close() did not close service")
	}
	if got := GetActiveAuthURLs(); len(got) != 0 {
		t.Fatalf("GetActiveAuthURLs() after Close = %#v, want empty", got)
	}
}
