package db

import "github.com/nbd-wtf/go-nostr"

func scanProfile(row scanner) (Profile, error) {
	var profile Profile
	err := row.Scan(
		&profile.ID,
		&profile.PublicKey,
		&profile.Name,
		&profile.About,
		&profile.Picture,
		&profile.Banner,
		&profile.Website,
		&profile.DisplayName,
		&profile.Lud16,
		&profile.Pronouns,
		&profile.Nip05,
		&profile.Bot,
	)
	return profile, err
}

func scanNostrEvent(row scanner) (*nostr.Event, error) {
	var evt nostr.Event
	var timestamp int64
	if err := row.Scan(
		&evt.ID,
		&evt.PubKey,
		&timestamp,
		&evt.Kind,
		&evt.Tags,
		&evt.Content,
		&evt.Sig,
	); err != nil {
		return nil, err
	}
	evt.CreatedAt = nostr.Timestamp(timestamp)
	return &evt, nil
}
