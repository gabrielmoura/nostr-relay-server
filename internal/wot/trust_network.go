package wot

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/nbd-wtf/go-nostr"
)

var (
	trustNetworkMap   = make(map[string]bool)
	trustNetworkMutex sync.RWMutex
	followerMutex     sync.RWMutex
	pool              *nostr.SimplePool

	lastComputed  time.Time
	recomputeChan = make(chan struct{}, 1)
)

type Summary struct {
	Nodes        int
	Edges        int // Not precisely tracked yet, but can estimate
	LastComputed time.Time
}

// Start initializes the background runner if Web of Trust is enabled.
func Start(ctx context.Context) {
	if !config.Cfg.WoT.Enabled || (config.Cfg.WoT.TargetPubkey == "" && len(config.Cfg.WoT.TrustedPubkeys) == 0) {
		return
	}

	log.Println("🌐 WOT subsystem enabled, initializing memory graph.")
	pool = nostr.NewSimplePool(ctx)

	// Start initial fetch
	go func() {
		refreshTrustNetwork(ctx)

		// Setup ticker based on config
		hours := config.Cfg.WoT.RefreshIntervalHours
		if hours <= 0 {
			hours = 3
		}
		ticker := time.NewTicker(time.Duration(hours) * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("🌐 WOT subsystem shutting down.")
				return
			case <-ticker.C:
				refreshTrustNetwork(ctx)
			case <-recomputeChan:
				log.Println("🌐 WOT recompute triggered manually.")
				refreshTrustNetwork(ctx)
			}
		}
	}()
}

// Validate checks if the pubkey exists in the trust network
func Validate(pubkey string) bool {
	// If WOT is disabled, all pubkeys are considered valid
	if !config.Cfg.WoT.Enabled || (config.Cfg.WoT.TargetPubkey == "" && len(config.Cfg.WoT.TrustedPubkeys) == 0) {
		return true
	}

	trustNetworkMutex.RLock()
	hasNetwork := len(trustNetworkMap) > 0
	trusted := trustNetworkMap[pubkey]
	trustNetworkMutex.RUnlock()

	// If the network map is empty, we are still booting up.
	// We safely allow everything through rather than blocking the relay.
	if !hasNetwork {
		return true
	}

	if !trusted {
		metrics.NostrBlockedWOTTotal.Inc()
	}

	return trusted
}

func ScheduleRecompute() {
	select {
	case recomputeChan <- struct{}{}:
	default:
		// Already scheduled
	}
}

func GetSummary() Summary {
	trustNetworkMutex.RLock()
	defer trustNetworkMutex.RUnlock()
	return Summary{
		Nodes:        len(trustNetworkMap),
		Edges:        0, // TODO: track edges if needed
		LastComputed: lastComputed,
	}
}

func refreshTrustNetwork(ctx context.Context) {
	newOneHopNetworkSet := make(map[string]bool)
	newPubkeyFollowerCount := make(map[string]int)
	var newOneHopNetwork []string

	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	roots := append([]string{}, config.Cfg.WoT.TrustedPubkeys...)
	if config.Cfg.WoT.TargetPubkey != "" {
		roots = append(roots, config.Cfg.WoT.TargetPubkey)
	}

	if len(roots) == 0 {
		return
	}

	filter := nostr.Filter{
		Authors: roots,
		Kinds:   []int{nostr.KindFollowList},
	}

	for ev := range pool.SubManyEose(timeoutCtx, config.Cfg.WoT.SeedRelays, nostr.Filters{filter}) {
		for _, tag := range ev.Tags {
			if len(tag) >= 2 && tag[0] == "p" {
				pubkey := tag[1]

				if len(pubkey) != 64 {
					continue
				}

				newPubkeyFollowerCount[pubkey]++

				if !newOneHopNetworkSet[pubkey] && len(newOneHopNetwork) < config.Cfg.WoT.MaxOneHopNetwork {
					newOneHopNetwork = append(newOneHopNetwork, pubkey)
					newOneHopNetworkSet[pubkey] = true
				}
			}
		}
	}

	// For standard configuration sizes, we can build the second degree list efficiently:
	if len(newOneHopNetwork) > 0 {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 10) // Concurrency limit

		// Fetch followers of the first layer
		for i := 0; i < len(newOneHopNetwork); i += 200 {
			end := i + 200
			if end > len(newOneHopNetwork) {
				end = len(newOneHopNetwork)
			}
			batch := newOneHopNetwork[i:end]

			wg.Add(1)
			sem <- struct{}{}
			go func(batch []string) {
				defer wg.Done()
				defer func() { <-sem }()

				fetchCtx, fetchCancel := context.WithTimeout(ctx, 60*time.Second)
				defer fetchCancel()

				batchFilter := nostr.Filter{
					Authors: batch,
					Kinds:   []int{nostr.KindFollowList},
				}

				for ev := range pool.SubManyEose(fetchCtx, config.Cfg.WoT.SeedRelays, nostr.Filters{batchFilter}) {
					for _, tag := range ev.Tags {
						if len(tag) >= 2 && tag[0] == "p" {
							pk := tag[1]
							if len(pk) == 64 {
								followerMutex.Lock()
								newPubkeyFollowerCount[pk]++
								followerMutex.Unlock()
							}
						}
					}
				}
			}(batch)
		}
		wg.Wait()
	}

	// Rebuild and swap map
	newTrustNetworkMap := make(map[string]bool)
	for _, root := range roots {
		newTrustNetworkMap[root] = true // Include the root targets natively
	}

	nodesAdded := 0
	for pubkey, count := range newPubkeyFollowerCount {
		if count >= config.Cfg.WoT.MinimumFollowers {
			newTrustNetworkMap[pubkey] = true
			nodesAdded++

			if config.Cfg.WoT.MaxTrustNetwork > 0 && nodesAdded >= config.Cfg.WoT.MaxTrustNetwork {
				break
			}
		}
	}

	trustNetworkMutex.Lock()
	trustNetworkMap = newTrustNetworkMap
	lastComputed = time.Now()
	trustNetworkMutex.Unlock()

	log.Printf("🌐 WOT cycle completed: Built graph containing %d trusted peers.", len(newTrustNetworkMap))
}
