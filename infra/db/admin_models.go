package db

import "database/sql"

type BannedUserRecord struct {
	Profile
	Reason     string       `json:"reason"`
	RelatedIDs []string     `json:"related_ids,omitempty"`
	CreatedAt  sql.NullTime `json:"created_at"`
}

type ReportedEventSummary struct {
	TargetEventID string   `json:"target_event_id"`
	TargetPubkey  string   `json:"target_pubkey"`
	ReportCount   int64    `json:"report_count"`
	LastReported  int64    `json:"last_reported"`
	ReportTypes   []string `json:"report_types"`
}

type ReportedTimelinePoint struct {
	Bucket string `json:"bucket"`
	Count  int64  `json:"count"`
}

type ReportedTypeCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type ReportedAuthorCount struct {
	Pubkey string `json:"pubkey"`
	Count  int64  `json:"count"`
}

type ReportedTargetCount struct {
	TargetEventID string `json:"target_event_id"`
	Count         int64  `json:"count"`
}

type ReportedEventsFilters struct {
	Query         string
	ReportType    string
	TargetPubkey  string
	TargetEventID string
	Since         int64
	Until         int64
}

type ReportedEventsSummary struct {
	TotalEvents         int64                   `json:"total_events"`
	TotalReports        int64                   `json:"total_reports"`
	UniqueTargetAuthors int64                   `json:"unique_target_authors"`
	Timeline            []ReportedTimelinePoint `json:"timeline"`
	ReportTypes         []ReportedTypeCount     `json:"report_types"`
	TopAuthors          []ReportedAuthorCount   `json:"top_authors"`
	TopTargets          []ReportedTargetCount   `json:"top_targets"`
}

type EventKindAggregate struct {
	Kind  int   `json:"kind"`
	Count int64 `json:"count"`
}

type EventAuthorAggregate struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"display_name,omitempty"`
	Count       int64  `json:"count"`
}

type EventTagAggregate struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type EventTrendAggregate struct {
	TopTagMonth      string `json:"top_tag_month,omitempty"`
	TopTagMonthCount int64  `json:"top_tag_month_count,omitempty"`
	TopTagYear       string `json:"top_tag_year,omitempty"`
	TopTagYearCount  int64  `json:"top_tag_year_count,omitempty"`
	PeakMonth        string `json:"peak_month,omitempty"`
	PeakMonthCount   int64  `json:"peak_month_count,omitempty"`
	PeakYear         string `json:"peak_year,omitempty"`
	PeakYearCount    int64  `json:"peak_year_count,omitempty"`
}

type EventTimelinePoint struct {
	TS    int64 `json:"ts"`
	Count int64 `json:"count"`
}

type EventAggregates struct {
	Total      int64                  `json:"total"`
	Kinds      []EventKindAggregate   `json:"kinds"`
	TopAuthors []EventAuthorAggregate `json:"top_authors"`
	TopTags    []EventTagAggregate    `json:"top_tags"`
	Trends     EventTrendAggregate    `json:"trends"`
}

type scanner interface {
	Scan(dest ...any) error
}
