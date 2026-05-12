package http

import (
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/handler/listener"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/stream"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/pkg/nostrpool"
	"github.com/gofiber/fiber/v2"
)

func AdminOverview() fiber.Handler {
	return func(c *fiber.Ctx) error {
		activeConnections := listener.ActiveConnections()
		authedConnections := listener.AuthedConnections()
		loggedUsers := aggregateLoggedUsers(authedConnections)

		bannedUsers, bannedTotal, err := db.DbQueries.ListBannedUsers(c.UserContext(), "", 1, 0)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return internalServerError(c, err)
		}
		_ = bannedUsers

		indexedEvents, err := db.DbQueries.CountAllEvents(c.UserContext())
		if err != nil {
			return internalServerError(c, err)
		}

		eventsPerMinute, err := db.DbQueries.CountEventsSince(c.UserContext(), timeNowUTCMinusMinute())
		if err != nil {
			return internalServerError(c, err)
		}

		return c.JSON(adminOverviewResponse{
			ActiveConnections: len(activeConnections),
			AuthedConnections: len(authedConnections),
			LoggedUsers:       len(loggedUsers),
			BannedUsers:       int(bannedTotal),
			IndexedEvents:     indexedEvents,
			EventsPerMinute:   eventsPerMinute,
			RelayStatus:       "operational",
		})
	}
}

func StreamStatus() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(adminStreamStatusResponse{
			Config: adminStreamConfigResponse{
				StreamUp:   config.Cfg.Stream.StreamUp,
				StreamDown: config.Cfg.Stream.StreamDown,
				Relays:     append([]string{}, config.Cfg.Stream.Relays...),
			},
			Dispatcher: stream.Snapshot(),
			Pool:       nostrpool.Stats(),
			Counters: adminStreamCountersResponse{
				ForwardedEvents:   int64(counterValue(metrics.NostrRelayEventForwardedTotal)),
				ForwardedRequests: int64(counterValue(metrics.NostrRelayRequestForwardedTotal)),
				ForwardFailures:   int64(counterValue(metrics.NostrRelayEventForwardedFailuresTotal)),
			},
		})
	}
}

func ActiveConnections() fiber.Handler {
	return func(c *fiber.Ctx) error {
		connections := listener.ActiveConnections()
		limit := adminLimit(c)
		offset := adminOffset(c)

		return c.JSON(newAdminPage(mapConnectionsPage(connections, limit, offset), len(connections), limit, offset))
	}
}

func AuthedConnections() fiber.Handler {
	return func(c *fiber.Ctx) error {
		connections := listener.AuthedConnections()
		limit := adminLimit(c)
		offset := adminOffset(c)

		return c.JSON(newAdminPage(mapConnectionsPage(connections, limit, offset), len(connections), limit, offset))
	}
}

func DisconnectConnection() fiber.Handler {
	return func(c *fiber.Ctx) error {
		wsid := c.Params("wsid")
		if wsid == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing wsid"})
		}

		var req DisconnectRequest
		if err := parseAdminJSONBody(c, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		_ = req
		if !listener.Disconnect(wsid) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "connection not found"})
		}
		return c.JSON(fiber.Map{"ws_id": wsid, "disconnected": true})
	}
}

func mapConnectionsPage(connections []listener.ConnectionInfo, limit int, offset int) []adminConnectionResponse {
	window := paginate(connections, limit, offset)
	items := make([]adminConnectionResponse, 0, len(window))
	for _, item := range window {
		items = append(items, adminConnectionResponse{
			WSID:              item.WSID,
			IP:                item.IP,
			Authed:            item.Authed,
			SubscriptionCount: item.SubscriptionCount,
			ConnectedAt:       formatUnix(item.ConnectedAt),
			LastSeenAt:        formatUnix(item.LastSeenAt),
			UserAgent:         item.UserAgent,
		})
	}
	return items
}

func aggregateLoggedUsers(connections []listener.ConnectionInfo) []loggedUserAggregate {
	grouped := make(map[string]loggedUserAggregate, len(connections))
	for _, connection := range connections {
		if connection.Authed == "" {
			continue
		}
		current := grouped[connection.Authed]
		current.Pubkey = connection.Authed
		current.ConnectionCount++
		if connection.LastSeenAt > current.LastSeenAt {
			current.LastSeenAt = connection.LastSeenAt
		}
		if connection.SubscriptionCount > 1 {
			current.ConnectionState = "stable"
		} else if current.ConnectionState == "" {
			current.ConnectionState = "attention"
		}
		grouped[connection.Authed] = current
	}

	users := make([]loggedUserAggregate, 0, len(grouped))
	for _, item := range grouped {
		if item.ConnectionState == "" {
			item.ConnectionState = "attention"
		}
		users = append(users, item)
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].ConnectionCount == users[j].ConnectionCount {
			if users[i].LastSeenAt == users[j].LastSeenAt {
				return users[i].Pubkey < users[j].Pubkey
			}
			return users[i].LastSeenAt > users[j].LastSeenAt
		}
		return users[i].ConnectionCount > users[j].ConnectionCount
	})

	return users
}

func newAdminPage[T any](items []T, total int, limit int, offset int) adminPage[T] {
	return adminPage[T]{
		Items:   items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+len(items) < total,
	}
}

func paginate[T any](items []T, limit int, offset int) []T {
	if offset >= len(items) {
		return []T{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func timeNowUTCMinusMinute() int64 {
	return time.Now().UTC().Add(-time.Minute).Unix()
}
