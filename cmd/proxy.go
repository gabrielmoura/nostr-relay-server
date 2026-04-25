package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/gabrielmoura/nostr-relay-server/config"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	proxyPort            int
	proxyOptimizeResults bool
	proxyPool            *nostr.SimplePool
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Starts a lightweight Nostr proxy instance",
	Long:  `Starts a Websocket router to act as a high-throughput event aggregator masking multiple upstream relays.`,
	Run:   runProxy,
}

func init() {
	proxyCmd.Flags().IntVarP(&proxyPort, "port", "p", 9092, "Port to run the proxy server on")
	proxyCmd.Flags().BoolVarP(&proxyOptimizeResults, "optimize-results", "o", true, "Enable server-side deduplication of events")
	proxyCmd.Flags().BoolP("config", "c", true, "Enable configuration file")
	rootCmd.AddCommand(proxyCmd)
}

func runProxy(cmd *cobra.Command, args []string) {
	if cmd.Flag("config").Value != nil {
		if err := config.LoadConfig(); err != nil {
			fmt.Println("Error loading config:", err)
		}
	}
	log.Init()

	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	relays := config.Cfg.Stream.Relays
	if len(relays) == 0 {
		log.Logger.Fatal("No upstream relays configured inside stream.relays for proxy operation.")
	}

	proxyPool = nostr.NewSimplePool(mainCtx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleProxyConn)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", proxyPort),
		Handler: mux,
	}

	go func() {
		<-stopChan
		log.Logger.Info("Shutting down proxy server...")
		server.Shutdown(mainCtx)
		mainCancel()
	}()

	log.Logger.Info("Starting Nostr Proxy Server",
		zap.Int("port", proxyPort),
		zap.Bool("deduplication", proxyOptimizeResults),
		zap.Strings("upstreams", relays),
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Logger.Fatal("Proxy server error", zap.Error(err))
	}
}

func handleProxyConn(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		log.Logger.Warn("WebSocket accept error", zap.Error(err))
		return
	}
	defer conn.Close(websocket.StatusInternalError, "internal error")

	ctx := r.Context()
	
	// Track active subscriptions for this client
	subs := make(map[string]context.CancelFunc)
	var subsMu sync.Mutex

	for {
		var rawMsg []json.RawMessage
		err = wsjson.Read(ctx, conn, &rawMsg)
		if err != nil {
			break
		}

		if len(rawMsg) < 2 {
			continue
		}

		var cmdType string
		if err := json.Unmarshal(rawMsg[0], &cmdType); err != nil {
			continue
		}

		switch cmdType {
		case "REQ":
			var subID string
			if err := json.Unmarshal(rawMsg[1], &subID); err != nil || subID == "" {
				continue
			}

			// Parse filters
			var filters nostr.Filters
			for _, rawFilter := range rawMsg[2:] {
				var filter nostr.Filter
				if err := json.Unmarshal(rawFilter, &filter); err == nil {
					filters = append(filters, filter)
				}
			}

			if len(filters) == 0 {
				continue
			}

			subCtx, cancel := context.WithCancel(ctx)
			
			subsMu.Lock()
			if oldCancel, exists := subs[subID]; exists {
				oldCancel() // Kill old subscription with same ID (NIP-01)
			}
			subs[subID] = cancel
			subsMu.Unlock()

			go processProxySubscription(subCtx, conn, subID, filters)

		case "CLOSE":
			var subID string
			if err := json.Unmarshal(rawMsg[1], &subID); err == nil {
				subsMu.Lock()
				if cancel, exists := subs[subID]; exists {
					cancel()
					delete(subs, subID)
				}
				subsMu.Unlock()
			}
			
		case "EVENT":
			var evt nostr.Event
			if err := json.Unmarshal(rawMsg[1], &evt); err == nil {
				// Relay incoming events to all upstreams
				for _, url := range config.Cfg.Stream.Relays {
					go func(u string) {
						if relay, err := nostr.RelayConnect(ctx, u); err == nil {
							_ = relay.Publish(ctx, evt)
							relay.Close()
						}
					}(url)
				}
				// Reply OK to client (blindly assuming success for simplicity in proxy mode)
				wsjson.Write(ctx, conn, []interface{}{"OK", evt.ID, true, ""})
			}
		}
	}
}

func processProxySubscription(ctx context.Context, conn *websocket.Conn, subID string, filters nostr.Filters) {
	seen := make(map[string]bool)
	relays := config.Cfg.Stream.Relays

	// SubMany returns a channel of nostr.IncomingEvent
	evCh := proxyPool.SubMany(ctx, relays, filters)

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-evCh:
			if !ok {
				return
			}
			
			if proxyOptimizeResults {
				if seen[ev.Event.ID] {
					continue
				}
				seen[ev.Event.ID] = true
			}
			
			msg := []interface{}{"EVENT", subID, ev.Event}
			_ = wsjson.Write(ctx, conn, msg)
		}
	}
}
