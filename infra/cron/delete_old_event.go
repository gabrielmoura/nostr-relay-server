package cron

import (
	"context"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"go.uber.org/zap"
	"strings"
	"time"
)

func DeleteOldEvent(before time.Time) {
	// delete events older than 1 month
	//before:=time.Now().AddDate(0, -1, 0)

	ctx := context.Background()

	events, err := db.DbQueries.GetOldEvents(ctx, before)
	if err != nil {
		return
	}
	total := len(events)
	totalDeleted := 0
	for _, event := range events {
		if err := db.DbQueries.DeleteEvent(ctx, event.ID); err == nil {
			totalDeleted++
		}
	}

	log.Logger.Info("Events deleted", zap.Int("total_deleted", totalDeleted), zap.Int("total", total))
}

func GenTime(timeStr string) time.Time {
	// timeStr = 1m, 1h, 1d, 1w, 1y, sendo a ultima letra a unidade de tempo
	// m = minute, h = hour, d = day, w = week, y = year
	var unit string
	var value int
	if len(timeStr) > 1 {
		unit = strings.ToLower(timeStr[len(timeStr)-1:])
		value = int(timeStr[0]) - 48
	} else {
		return time.Time{}
	}

	switch unit {
	case "m":
		return time.Now().Add(time.Duration(value) * time.Minute)
	case "h":
		return time.Now().Add(time.Duration(value) * time.Hour)
	case "d":
		return time.Now().AddDate(0, 0, value)
	case "w":
		return time.Now().AddDate(0, 0, value*7)
	case "y":
		return time.Now().AddDate(value, 0, 0)
	default:
		return time.Time{}
	}

}
