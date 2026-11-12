package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nbd-wtf/go-nostr"
)

type timeBucket struct {
	Day         string `json:"day,omitempty"`
	Hour        string `json:"hour,omitempty"`
	Count       int    `json:"count"`
	Approximate bool   `json:"approximate"`
}

type monthBucket struct {
	Month       string `json:"month"`
	Count       int    `json:"count"`
	Approximate bool   `json:"approximate"`
}

func (h *Handler) authorMonthly(c *fiber.Ctx) error {
	pubkey, err := decodeAuthor(c.Params("npub"))
	if err != nil {
		return badRequest(c, err)
	}
	kind, err := strconv.Atoi(c.Params("kind"))
	if err != nil || kind < 0 {
		return badRequest(c, errors.New("kind must be a non-negative integer"))
	}
	since, err := timestamp(c.Query("since"))
	if err != nil {
		return badRequest(c, errors.New("invalid since"))
	}
	until, err := timestamp(c.Query("until"))
	if err != nil {
		return badRequest(c, errors.New("invalid until"))
	}
	if until == nil {
		now := nostr.Timestamp(time.Now().Unix())
		until = &now
	}
	filter := nostr.Filter{Authors: []string{pubkey}, Kinds: []int{kind}}
	if since == nil {
		events, exhausted, err := h.scan(c.UserContext(), filter, 0, 1, excludedTags(c), true)
		if err != nil {
			return internalError(c, err)
		}
		if len(events) == 0 {
			if !exhausted {
				return internalError(c, errScanLimit)
			}
			return c.JSON(fiber.Map{"months": []monthBucket{}, "total": 0, "approximate": false})
		}
		first := events[0].CreatedAt
		since = &first
	}
	if *since > *until {
		return badRequest(c, errors.New("until must not be earlier than since"))
	}
	firstTime, lastTime := time.Unix(int64(*since), 0).UTC(), time.Unix(int64(*until), 0).UTC()
	if firstTime.Year() < 1970 || lastTime.Year() > 9999 {
		return badRequest(c, errors.New("timestamp is outside the supported range"))
	}
	months := (lastTime.Year()-firstTime.Year())*12 + int(lastTime.Month()-firstTime.Month()) + 1
	if months > 120 {
		return badRequest(c, errors.New("the requested range exceeds 120 months"))
	}
	start := time.Date(firstTime.Year(), firstTime.Month(), 1, 0, 0, 0, 0, time.UTC)
	buckets := make([]monthBucket, 0, months)
	total, approximate := 0, false
	for i := 0; i < months; i++ {
		begin := start.AddDate(0, i, 0)
		monthStart, monthEnd := nostr.Timestamp(begin.Unix()), nostr.Timestamp(begin.AddDate(0, 1, 0).Unix()-1)
		if monthStart < *since {
			monthStart = *since
		}
		if monthEnd > *until {
			monthEnd = *until
		}
		window := filter
		window.Since, window.Until = &monthStart, &monthEnd
		events, exhausted, err := h.scan(c.UserContext(), window, 0, h.maxScan(), excludedTags(c), false)
		if err != nil {
			return internalError(c, err)
		}
		bucket := monthBucket{Month: begin.Format("2006-01"), Count: len(events), Approximate: !exhausted}
		buckets = append(buckets, bucket)
		total += bucket.Count
		approximate = approximate || bucket.Approximate
	}
	return c.JSON(fiber.Map{"months": buckets, "total": total, "approximate": approximate})
}

func datePart(c *fiber.Ctx, key string, fallback, minValue, maxValue int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minValue || parsed > maxValue {
		return 0, errors.New("invalid " + key)
	}
	return parsed, nil
}

func (h *Handler) authorDaily(c *fiber.Ctx) error  { return h.authorTimeSeries(c, false) }
func (h *Handler) authorHourly(c *fiber.Ctx) error { return h.authorTimeSeries(c, true) }

func (h *Handler) authorTimeSeries(c *fiber.Ctx, hourly bool) error {
	pubkey, err := decodeAuthor(c.Params("npub"))
	if err != nil {
		return badRequest(c, err)
	}
	kind, err := strconv.Atoi(c.Params("kind"))
	if err != nil || kind < 0 {
		return badRequest(c, errors.New("kind must be a non-negative integer"))
	}
	if c.Query("since") != "" || c.Query("until") != "" {
		return badRequest(c, errors.New("since and until are not supported for this endpoint"))
	}
	now := time.Now().UTC()
	year, err := datePart(c, "year", now.Year(), 1970, 2999)
	if err != nil {
		return badRequest(c, err)
	}
	month, err := datePart(c, "month", int(now.Month()), 1, 12)
	if err != nil {
		return badRequest(c, err)
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	days := start.AddDate(0, 1, 0).AddDate(0, 0, -1).Day()
	count, step, label := days, 24*time.Hour, "days"
	if hourly {
		defaultDay := 1
		if year == now.Year() && time.Month(month) == now.Month() {
			defaultDay = now.Day()
		}
		day, err := datePart(c, "day", defaultDay, 1, days)
		if err != nil {
			return badRequest(c, err)
		}
		start = start.AddDate(0, 0, day-1)
		count, step, label = 24, time.Hour, "hours"
	}
	buckets := make([]timeBucket, 0, count)
	total := 0
	approximate := false
	for i := 0; i < count; i++ {
		begin := start.Add(time.Duration(i) * step)
		since, until := nostr.Timestamp(begin.Unix()), nostr.Timestamp(begin.Add(step).Unix()-1)
		filter := nostr.Filter{Authors: []string{pubkey}, Kinds: []int{kind}, Since: &since, Until: &until}
		events, exhausted, err := h.scan(c.UserContext(), filter, 0, h.maxScan(), excludedTags(c), false)
		if err != nil {
			return internalError(c, err)
		}
		bucket := timeBucket{Count: len(events), Approximate: !exhausted}
		if hourly {
			bucket.Hour = begin.Format("2006-01-02T15")
		} else {
			bucket.Day = begin.Format("2006-01-02")
		}
		buckets = append(buckets, bucket)
		total += bucket.Count
		approximate = approximate || bucket.Approximate
	}
	return c.JSON(fiber.Map{label: buckets, "total": total, "approximate": approximate})
}
