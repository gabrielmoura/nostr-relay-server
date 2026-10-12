package fiber

import (
	"context"
	"net"
	"time"

	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/protobuf/proto"
)

type LiveOptions struct {
	NewSync       func(context.Context, string, *relaysyncv2.ClientHello) (*core.LiveSync, error)
	MaxFrameBytes int
}

func RegisterLive(app *fiber.App, options LiveOptions) error {
	if app == nil || options.NewSync == nil || options.MaxFrameBytes <= 0 {
		return core.ErrInvalidLimits
	}
	app.Get("/sync/v2/live", websocket.New(func(conn *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		frame := &relaysyncv2.ClientFrame{}
		if proto.Unmarshal(payload, frame) != nil || frame.GetHello() == nil || frame.GetHello().ProtocolVersion != 2 {
			return
		}
		peer := conn.RemoteAddr().String()
		if host, _, splitErr := net.SplitHostPort(peer); splitErr == nil {
			peer = host
		}
		live, err := options.NewSync(ctx, peer, frame.GetHello())
		if err != nil {
			return
		}
		defer live.Close()
		_ = writeLive(conn, &relaysyncv2.ServerFrame{Body: &relaysyncv2.ServerFrame_Hello{Hello: &relaysyncv2.ServerHello{ProtocolVersion: 2, Limits: &relaysyncv2.Limits{MaxFrameBytes: uint32(options.MaxFrameBytes)}}}})
		if err := live.Start(ctx, cursorFromProto(frame.GetHello().ResumeCursor)); err != nil {
			return
		}
		for {
			if _, payload, err = conn.ReadMessage(); err != nil {
				return
			}
			if len(payload) > options.MaxFrameBytes {
				return
			}
			frame = &relaysyncv2.ClientFrame{}
			if proto.Unmarshal(payload, frame) != nil {
				return
			}
			if flow := frame.GetFlowControl(); flow != nil {
				if live.SetFlowControl(core.FlowControl{MaxEvents: int(flow.MaxEvents), MaxBytes: int(flow.MaxBytes), Paused: flow.Pause}) != nil {
					return
				}
			}
			if ack := frame.GetAck(); ack != nil {
				if live.Ack(ctx, cursorFromProto(ack.Cursor), "fiber-ws") != nil {
					return
				}
			}
			if frame.GetHeartbeat() != nil {
				live.ObserveHeartbeat(time.Now())
			}
			_, _ = live.Pump(ctx, 64)
			if batch, err := live.NextBatch(); err == nil {
				if writeLive(conn, &relaysyncv2.ServerFrame{Body: &relaysyncv2.ServerFrame_Delta{Delta: deltaProto(batch)}}) != nil {
					return
				}
			}
		}
	}))
	return nil
}

func writeLive(conn *websocket.Conn, frame proto.Message) error {
	payload, err := proto.Marshal(frame)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.BinaryMessage, payload)
}
func cursorFromProto(cursor *relaysyncv2.Cursor) core.Cursor {
	if cursor == nil {
		return core.Cursor{}
	}
	return core.Cursor{Epoch: cursor.Epoch, Sequence: cursor.Sequence}
}
func deltaProto(batch core.LiveBatch) *relaysyncv2.DeltaBatch {
	result := &relaysyncv2.DeltaBatch{FirstCursor: &relaysyncv2.Cursor{Epoch: batch.First.Epoch, Sequence: batch.First.Sequence}, LastCursor: &relaysyncv2.Cursor{Epoch: batch.Last.Epoch, Sequence: batch.Last.Sequence}}
	for _, change := range batch.Changes {
		item := &relaysyncv2.JournalChange{Cursor: &relaysyncv2.Cursor{Epoch: change.Cursor.Epoch, Sequence: change.Cursor.Sequence}, EventId: append([]byte(nil), change.ID[:]...)}
		if change.Type == core.ChangeUpsert {
			item.ChangeType = 1
			item.Event = core.EventToProto(*change.Event)
		} else {
			item.ChangeType = 2
		}
		result.Changes = append(result.Changes, item)
	}
	return result
}
