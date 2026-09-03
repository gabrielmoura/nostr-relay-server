package graph

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gabrielmoura/nostr-relay-server/graph/model"
	httphandler "github.com/gabrielmoura/nostr-relay-server/infra/handler/http"
)

func (r *Resolver) adminOverview(ctx context.Context) (*model.AdminOverview, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/overview", path: "/overview", handlerFunc: httphandler.AdminOverview()})
	if err != nil {
		return nil, err
	}
	return decodeRESTModel[model.AdminOverview](payload)
}

func (r *Resolver) adminStreamStatus(ctx context.Context) (*model.AdminStreamStatus, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/stream/status", path: "/stream/status", handlerFunc: httphandler.StreamStatus()})
	if err != nil {
		return nil, err
	}
	return decodeRESTModel[model.AdminStreamStatus](payload)
}

func (r *Resolver) activeConnections(ctx context.Context, page *model.OffsetPageInput) (*model.AdminConnectionPage, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/connections/active", path: "/connections/active", query: adminPageQuery(page), handlerFunc: httphandler.ActiveConnections()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.AdminConnection](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminConnectionPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) authedConnections(ctx context.Context, page *model.OffsetPageInput) (*model.AdminConnectionPage, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/connections/authed", path: "/connections/authed", query: adminPageQuery(page), handlerFunc: httphandler.AuthedConnections()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.AdminConnection](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminConnectionPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) loggedUsers(ctx context.Context, page *model.OffsetPageInput) (*model.AdminLoggedUserPage, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/users/logged", path: "/users/logged", query: adminPageQuery(page), handlerFunc: httphandler.LoggedUsers()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.AdminLoggedUser](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminLoggedUserPage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) bannedUsers(ctx context.Context, q *string, page *model.OffsetPageInput) (*model.AdminProfilePage, error) {
	query := adminPageQuery(page)
	if value := strings.TrimSpace(stringValue(q)); value != "" {
		query.Set("q", value)
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/users/banned", path: "/users/banned", query: query, handlerFunc: httphandler.BannedUsers()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.AdminProfile](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminProfilePage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) searchUsers(ctx context.Context, q *string, page *model.OffsetPageInput) (*model.AdminProfilePage, error) {
	query := adminPageQuery(page)
	if value := strings.TrimSpace(stringValue(q)); value != "" {
		query.Set("q", value)
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/users/search", path: "/users/search", query: query, handlerFunc: httphandler.SearchUsers()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.AdminProfile](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminProfilePage{PageInfo: pageInfo, Items: items}, nil
}

func (r *Resolver) userProfile(ctx context.Context, pubkey string) (*model.AdminProfile, error) {
	path := fmt.Sprintf("/users/%s/profile", pubkey)
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/users/:pubkey/profile", path: path, handlerFunc: httphandler.UserProfile()})
	if err != nil {
		return nil, err
	}
	return decodeRESTModel[model.AdminProfile](payload)
}

func (r *Resolver) userBanStatus(ctx context.Context, pubkey string) (*model.AdminBanStatus, error) {
	path := fmt.Sprintf("/users/%s/ban", pubkey)
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/users/:pubkey/ban", path: path, handlerFunc: httphandler.BanStatus()})
	if err != nil {
		return nil, err
	}
	return decodeRESTModel[model.AdminBanStatus](payload)
}

func (r *Resolver) userNip05(ctx context.Context, pubkey string) (*model.AdminNip05Lookup, error) {
	path := fmt.Sprintf("/users/%s/nip05", pubkey)
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/users/:pubkey/nip05", path: path, handlerFunc: httphandler.UserNIP05()})
	if err != nil {
		return nil, err
	}
	lookup, err := decodeRESTModel[model.AdminNip05Lookup](payload)
	if err != nil {
		return nil, err
	}
	if lookup.Exists && lookup.Identity == nil {
		identity, err := decodeRESTModel[model.AdminNip05Identity](payload)
		if err == nil {
			lookup.Identity = identity
		}
	}
	return lookup, nil
}

func (r *Resolver) nip05Identities(ctx context.Context, q *string, page *model.OffsetPageInput) (*model.AdminNip05IdentityPage, error) {
	query := adminPageQuery(page)
	if value := strings.TrimSpace(stringValue(q)); value != "" {
		query.Set("q", value)
	}
	payload, err := executeAdminRequest(ctx, adminRequest{method: http.MethodGet, route: "/nip05", path: "/nip05", query: query, handlerFunc: httphandler.NIP05List()})
	if err != nil {
		return nil, err
	}
	items, pageInfo, err := decodeRESTPage[model.AdminNip05Identity](payload)
	if err != nil {
		return nil, err
	}
	return &model.AdminNip05IdentityPage{PageInfo: pageInfo, Items: items}, nil
}

func buildEventSearchQuery(filter *model.AdminEventSearchInput, page *model.OffsetPageInput) url.Values {
	query := adminPageQuery(page)
	if filter == nil {
		return query
	}
	if value := strings.TrimSpace(stringValue(filter.Q)); value != "" {
		query.Set("q", value)
	}
	for _, author := range filter.Authors {
		query.Add("author", author)
	}
	for _, kind := range filter.Kinds {
		query.Add("kind", fmt.Sprintf("%d", kind))
	}
	for _, tag := range filter.Tags {
		if tag == nil {
			continue
		}
		query.Add("tag", tag.Name+":"+tag.Value)
	}
	if filter.Since != nil {
		query.Set("since", fmt.Sprintf("%d", *filter.Since))
	}
	if filter.Until != nil {
		query.Set("until", fmt.Sprintf("%d", *filter.Until))
	}
	return query
}

// privacyStatus returns the aggregated privacy-network observability snapshot
// (Tor / I2P / Yggdrasil) for the admin dashboard. Delegates to the internal
// /privacy/status HTTP handler, mirroring adminStreamStatus.
func (r *Resolver) privacyStatus(ctx context.Context) (*model.PrivacyStatus, error) {
	payload, err := executeAdminRequest(ctx, adminRequest{
		method:      http.MethodGet,
		route:       "/privacy/status",
		path:        "/privacy/status",
		handlerFunc: httphandler.PrivacyStatus(),
	})
	if err != nil {
		return nil, err
	}
	return decodeRESTModel[model.PrivacyStatus](payload)
}
