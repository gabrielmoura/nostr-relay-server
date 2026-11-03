package graph

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/gabrielmoura/nostr-relay-server/graph/model"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	internaldb "github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// LoggedUsersSource supplies the data behind the GraphQL loggedUsers query.
// It keeps the resolver independent from the administrative HTTP handlers.
type LoggedUsersSource interface {
	LoggedUsers(context.Context, *model.OffsetPageInput, bool) (*model.AdminLoggedUserPage, error)
}

type runtimeLoggedUsersSource struct{}

type loggedUserAggregate struct {
	pubkey          string
	connectionCount int32
	lastSeenAt      int64
	connectionState string
}

func (runtimeLoggedUsersSource) LoggedUsers(ctx context.Context, page *model.OffsetPageInput, includeProfiles bool) (*model.AdminLoggedUserPage, error) {
	users := aggregateRuntimeLoggedUsers(listener.AuthedConnections())
	limit, offset := loggedUsersPage(page)
	window := pageLoggedUsers(users, limit, offset)

	profiles := map[string]profileRecord{}
	if includeProfiles && len(window) > 0 {
		if internaldb.DbQueries == nil {
			return nil, errors.New("profile query service is not initialized")
		}
		pubkeys := make([]string, 0, len(window))
		for _, user := range window {
			pubkeys = append(pubkeys, user.pubkey)
		}
		stored, err := internaldb.DbQueries.GetProfilesByPublicKeys(ctx, pubkeys)
		if err != nil {
			return nil, err
		}
		for pubkey, profile := range stored {
			profiles[pubkey] = profileRecord{
				pubkey: profile.PublicKey, name: profile.Name, displayName: profile.DisplayName,
				picture: profile.Picture, nip05: profile.Nip05, about: profile.About,
				website: profile.Website, bot: profile.Bot,
			}
		}
	}

	items := make([]*model.AdminLoggedUser, 0, len(window))
	for _, user := range window {
		item := &model.AdminLoggedUser{
			ConnectionCount: user.connectionCount,
			ConnectionState: user.connectionState,
		}
		if user.lastSeenAt > 0 {
			value := time.Unix(user.lastSeenAt, 0).UTC().Format(time.RFC3339)
			item.LastSeenAt = &value
		}
		if includeProfiles {
			item.Profile = profileToLoggedUserProfile(profiles[user.pubkey], user.pubkey)
		}
		items = append(items, item)
	}

	return &model.AdminLoggedUserPage{
		PageInfo: &model.PageInfo{Total: int32(len(users)), Limit: int32(limit), Offset: int32(offset), HasMore: offset+len(items) < len(users)},
		Items:    items,
	}, nil
}

type profileRecord struct {
	pubkey, name, displayName, picture, nip05, about, website string
	bot                                                       bool
}

func profileToLoggedUserProfile(profile profileRecord, fallbackPubkey string) *model.AdminProfile {
	pubkey := profile.pubkey
	if pubkey == "" {
		pubkey = fallbackPubkey
	}
	displayName := strings.TrimSpace(profile.displayName)
	if displayName == "" {
		displayName = strings.TrimSpace(profile.name)
	}
	if displayName == "" {
		displayName = pubkey
	}
	handle := strings.ReplaceAll(strings.TrimPrefix(strings.TrimSpace(profile.name), "@"), " ", "_")
	return &model.AdminProfile{
		Pubkey: pubkey, Npub: optionalNpub(pubkey), DisplayName: displayName,
		Handle: optionalString(handle), Picture: optionalString(profile.picture), Nip05: optionalString(profile.nip05),
		About: optionalString(profile.about), Website: optionalString(profile.website), Bot: profile.bot,
		Status: optionalString("online"), RelatedIds: []string{},
	}
}

func optionalNpub(pubkey string) *string {
	npub, err := nip19.EncodePublicKey(pubkey)
	if err != nil || npub == "" {
		return nil
	}
	return &npub
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func aggregateRuntimeLoggedUsers(connections []listener.ConnectionInfo) []loggedUserAggregate {
	grouped := make(map[string]loggedUserAggregate, len(connections))
	for _, connection := range connections {
		if connection.Authed == "" {
			continue
		}
		user := grouped[connection.Authed]
		user.pubkey = connection.Authed
		user.connectionCount++
		if connection.LastSeenAt > user.lastSeenAt {
			user.lastSeenAt = connection.LastSeenAt
		}
		if connection.SubscriptionCount > 1 {
			user.connectionState = "stable"
		} else if user.connectionState == "" {
			user.connectionState = "attention"
		}
		grouped[user.pubkey] = user
	}
	users := make([]loggedUserAggregate, 0, len(grouped))
	for _, user := range grouped {
		if user.connectionState == "" {
			user.connectionState = "attention"
		}
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool {
		if users[i].connectionCount == users[j].connectionCount {
			if users[i].lastSeenAt == users[j].lastSeenAt {
				return users[i].pubkey < users[j].pubkey
			}
			return users[i].lastSeenAt > users[j].lastSeenAt
		}
		return users[i].connectionCount > users[j].connectionCount
	})
	return users
}

func loggedUsersPage(page *model.OffsetPageInput) (limit, offset int) {
	limit = 100
	if page != nil && page.Limit != nil && *page.Limit > 0 {
		limit = min(int(*page.Limit), 250)
	}
	if page != nil && page.Offset != nil && *page.Offset > 0 {
		offset = int(*page.Offset)
	}
	return limit, offset
}

func pageLoggedUsers(users []loggedUserAggregate, limit, offset int) []loggedUserAggregate {
	if offset >= len(users) {
		return []loggedUserAggregate{}
	}
	end := min(offset+limit, len(users))
	return users[offset:end]
}

func loggedUserProfileSelected(ctx context.Context) bool {
	field := graphql.GetFieldContext(ctx)
	operation := graphql.GetOperationContext(ctx)
	if field == nil || operation == nil {
		return true
	}
	for _, child := range graphql.CollectFields(operation, field.Field.Selections, nil) {
		if child.Name == "items" {
			for _, itemField := range graphql.CollectFields(operation, child.Selections, nil) {
				if itemField.Name == "profile" {
					return true
				}
			}
		}
	}
	return false
}
