package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/infra/cache"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	json "github.com/gabrielmoura/nostr-relay-server/internal/jsonx"
	"github.com/nbd-wtf/go-nostr"

	dbstore "github.com/gabrielmoura/nostr-relay-server/infra/db"
)

func (m *Manager) getGroupFromCache(groupID string) (*dbstore.NIP29Group, bool) {
	if !m.cfg.Advanced.CacheGroupMetadata || !cache.IsEnabled() {
		return nil, false
	}
	value, err := cache.GetRedis().Get(context.Background(), m.groupKey(groupID))
	if err != nil || value == "" {
		metrics.NostrNIP29CacheTotal.WithLabelValues("group", "miss").Inc()
		return nil, false
	}
	var group dbstore.NIP29Group
	if err := json.Unmarshal([]byte(value), &group); err != nil {
		metrics.NostrNIP29CacheTotal.WithLabelValues("group", "miss").Inc()
		return nil, false
	}
	metrics.NostrNIP29CacheTotal.WithLabelValues("group", "hit").Inc()
	return &group, true
}

func (m *Manager) storeGroupCache(group *dbstore.NIP29Group) {
	if group == nil || !m.cfg.Advanced.CacheGroupMetadata || !cache.IsEnabled() {
		return
	}
	data, err := json.Marshal(group)
	if err != nil {
		return
	}
	ttl := time.Duration(maxInt(m.cfg.CacheTTLSeconds, 60)) * time.Second
	_ = cache.GetRedis().Set(context.Background(), m.groupKey(group.GroupID), string(data), ttl)
}

func (m *Manager) getMembershipCache(groupID, pubkey string) (bool, bool) {
	if !m.cfg.Advanced.CacheMembershipLookup || !cache.IsEnabled() {
		return false, false
	}
	value, err := cache.GetRedis().Get(context.Background(), m.memberKey(groupID, pubkey))
	if err != nil {
		metrics.NostrNIP29CacheTotal.WithLabelValues("member", "miss").Inc()
		return false, false
	}
	metrics.NostrNIP29CacheTotal.WithLabelValues("member", "hit").Inc()
	return value == "1", true
}

func (m *Manager) storeMembershipCache(groupID, pubkey string, exists bool) {
	if !m.cfg.Advanced.CacheMembershipLookup || !cache.IsEnabled() {
		return
	}
	value := "0"
	if exists {
		value = "1"
	}
	ttl := time.Duration(maxInt(m.cfg.MembershipCacheTTLSeconds, 30)) * time.Second
	_ = cache.GetRedis().Set(context.Background(), m.memberKey(groupID, pubkey), value, ttl)
}

func (m *Manager) getBanCache(groupID, pubkey string) (bool, string, bool) {
	if !cache.IsEnabled() {
		return false, "", false
	}
	value, err := cache.GetRedis().Get(context.Background(), m.banKey(groupID, pubkey))
	if err != nil {
		metrics.NostrNIP29CacheTotal.WithLabelValues("ban", "miss").Inc()
		return false, "", false
	}
	metrics.NostrNIP29CacheTotal.WithLabelValues("ban", "hit").Inc()
	if value == "" || value == "0" {
		return false, "", true
	}
	return true, value, true
}

func (m *Manager) storeBanCache(groupID, pubkey string, exists bool, reason string) {
	if !cache.IsEnabled() {
		return
	}
	value := "0"
	if exists {
		value = reason
		if value == "" {
			value = "banned"
		}
	}
	ttl := time.Duration(maxInt(m.cfg.BanCacheTTLSeconds, 30)) * time.Second
	_ = cache.GetRedis().Set(context.Background(), m.banKey(groupID, pubkey), value, ttl)
}

func (m *Manager) getInviteCache(groupID, code string) (*dbstore.NIP29Invite, bool) {
	if !cache.IsEnabled() || !m.cfg.Invite.Enabled {
		return nil, false
	}
	value, err := cache.GetRedis().Get(context.Background(), m.inviteKey(groupID, code))
	if err != nil || value == "" {
		metrics.NostrNIP29CacheTotal.WithLabelValues("invite", "miss").Inc()
		return nil, false
	}
	var invite dbstore.NIP29Invite
	if err := json.Unmarshal([]byte(value), &invite); err != nil {
		metrics.NostrNIP29CacheTotal.WithLabelValues("invite", "miss").Inc()
		return nil, false
	}
	metrics.NostrNIP29CacheTotal.WithLabelValues("invite", "hit").Inc()
	return &invite, true
}

func (m *Manager) storeInviteCache(invite *dbstore.NIP29Invite) {
	if invite == nil || !cache.IsEnabled() || !m.cfg.Invite.Enabled {
		return
	}
	data, err := json.Marshal(invite)
	if err != nil {
		return
	}
	ttl := time.Duration(maxInt(m.cfg.Invite.DefaultTTLSeconds, 60)) * time.Second
	if invite.ExpiresAt != nil {
		ttl = time.Until(*invite.ExpiresAt)
		if ttl <= 0 {
			return
		}
	}
	_ = cache.GetRedis().Set(context.Background(), m.inviteKey(invite.GroupID, invite.Code), string(data), ttl)
}

func (m *Manager) recordTimelineReference(ctx context.Context, evt *nostr.Event) {
	groupID := groupIDFromEvent(evt)
	if groupID == "" {
		return
	}
	redisClient := cache.GetRedis()
	if redisClient == nil || !cache.IsEnabled() || !m.cfg.Timeline.Enabled {
		return
	}
	raw := redisClient.Raw()
	if raw == nil {
		return
	}

	window := int64(maxInt(m.cfg.Timeline.RecentWindow, 50) - 1)
	ttl := time.Duration(maxInt(m.cfg.TimelineCacheTTLSeconds, 300)) * time.Second
	pipe := raw.Pipeline()
	pipe.LPush(ctx, m.timelineKey(groupID), evt.ID)
	pipe.LTrim(ctx, m.timelineKey(groupID), 0, window)
	pipe.Expire(ctx, m.timelineKey(groupID), ttl)
	_, _ = pipe.Exec(ctx)
	metrics.NostrNIP29CacheTotal.WithLabelValues("timeline", "write").Inc()
}

func (m *Manager) getTimelineFromCache(groupID string, window int) []string {
	redisClient := cache.GetRedis()
	if redisClient == nil || !cache.IsEnabled() || !m.cfg.Timeline.Enabled {
		return nil
	}
	values, err := redisClient.Raw().LRange(context.Background(), m.timelineKey(groupID), 0, int64(window-1)).Result()
	if err != nil || len(values) == 0 {
		metrics.NostrNIP29CacheTotal.WithLabelValues("timeline", "miss").Inc()
		return nil
	}
	metrics.NostrNIP29CacheTotal.WithLabelValues("timeline", "hit").Inc()
	return values
}

func (m *Manager) invalidateGroupCaches(groupID string) {
	if !cache.IsEnabled() {
		return
	}
	_ = cache.GetRedis().Del(context.Background(), m.groupKey(groupID))
}

func (m *Manager) invalidateMemberCache(groupID, pubkey string) {
	if !cache.IsEnabled() {
		return
	}
	_ = cache.GetRedis().Del(context.Background(), m.memberKey(groupID, pubkey), m.banKey(groupID, pubkey))
}

func (m *Manager) invalidateInviteCache(groupID, code string) {
	if !cache.IsEnabled() {
		return
	}
	_ = cache.GetRedis().Del(context.Background(), m.inviteKey(groupID, code))
}

func (m *Manager) groupKey(groupID string) string {
	return fmt.Sprintf("nip29:group:%s:%s", m.relayScope, groupID)
}

func (m *Manager) memberKey(groupID, pubkey string) string {
	return fmt.Sprintf("nip29:member:%s:%s:%s", m.relayScope, groupID, pubkey)
}

func (m *Manager) banKey(groupID, pubkey string) string {
	return fmt.Sprintf("nip29:ban:%s:%s:%s", m.relayScope, groupID, pubkey)
}

func (m *Manager) inviteKey(groupID, code string) string {
	return fmt.Sprintf("nip29:invite:%s:%s:%s", m.relayScope, groupID, code)
}

func (m *Manager) timelineKey(groupID string) string {
	return fmt.Sprintf("nip29:timeline:%s:%s", m.relayScope, groupID)
}
