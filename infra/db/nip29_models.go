package db

import "time"

type NIP29Group struct {
	Relay                        string
	GroupID                      string
	Name                         string
	Picture                      string
	About                        string
	Private                      bool
	Closed                       bool
	Restricted                   bool
	Hidden                       bool
	CreatedBy                    string
	UpdatedAt                    time.Time
	DeletedAt                    *time.Time
	MinPoW                       int
	RequireModerationTimelineRef bool
	MinTimelineReferences        int
	TimelineRecentWindow         int
	AllowLatePublication         bool
	LastMetadataUpdate           time.Time
	LastAdminsUpdate             time.Time
	LastMembersUpdate            time.Time
	LastRolesUpdate              time.Time
}

type NIP29Role struct {
	RoleID      int32
	Name        string
	Description string
}

type NIP29MemberRole struct {
	UserID      string
	RoleID      int32
	RoleName    string
	Description string
}

type NIP29Invite struct {
	Relay      string
	GroupID    string
	Code       string
	CreatedBy  string
	MaxUses    int
	Uses       int
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	LastUsedAt *time.Time
}
