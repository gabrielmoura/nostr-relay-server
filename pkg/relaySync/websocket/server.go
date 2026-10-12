// Package websocket provides the binary RelaySync live transport adapter.
// TLS, HTTP/2 and listener lifecycle remain responsibilities of the host.
package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
	core "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync"
	relaysyncv2 "github.com/gabrielmoura/nostr-relay-server/pkg/relaySync/proto/relaysync/v2"
	"google.golang.org/protobuf/proto"
)

const Subprotocol = "nostr-sync.v2.pb"

type Options struct {
	NewSync           func(context.Context, string, *relaysyncv2.ClientHello) (*core.LiveSync, error)
	Authorize         func(*http.Request, *relaysyncv2.ClientHello) (string, bool)
	MaxFrameBytes     int
	HeartbeatInterval time.Duration
}

type Server struct{ options Options }

func NewServer(options Options) (*Server, error) {
	if options.NewSync == nil || options.Authorize == nil || options.MaxFrameBytes <= 0 {
		return nil, core.ErrInvalidLimits
	}
	if options.HeartbeatInterval <= 0 {
		options.HeartbeatInterval = 15 * time.Second
	}
	return &Server{options: options}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{Subprotocol}, CompressionMode: websocket.CompressionDisabled})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	_, payload, err := conn.Read(ctx)
	if err != nil {
		return
	}
	var frame relaysyncv2.ClientFrame
	if err := proto.Unmarshal(payload, &frame); err != nil {
		_ = writeError(ctx, conn, relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_INVALID_MESSAGE, "invalid protobuf")
		return
	}
	hello := frame.GetHello()
	if hello == nil || hello.ProtocolVersion != 2 {
		_ = writeError(ctx, conn, relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_INVALID_MESSAGE, "client hello required")
		return
	}
	peer, ok := s.options.Authorize(r, hello)
	if !ok {
		_ = writeError(ctx, conn, relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_UNAUTHORIZED, "peer not allowed")
		return
	}
	syncer, err := s.options.NewSync(ctx, peer, hello)
	if err != nil {
		_ = writeError(ctx, conn, relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_INTERNAL, err.Error())
		return
	}
	defer syncer.Close()
	if err := writeFrame(ctx, conn, &relaysyncv2.ServerFrame{Body: &relaysyncv2.ServerFrame_Hello{Hello: &relaysyncv2.ServerHello{ProtocolVersion: 2, Compression: relaysyncv2.Compression_COMPRESSION_NONE, Limits: &relaysyncv2.Limits{MaxFrameBytes: uint32(s.options.MaxFrameBytes)}}}}); err != nil {
		return
	}
	if err := syncer.Start(ctx, cursorFromProto(hello.ResumeCursor)); err != nil {
		_ = writeError(ctx, conn, relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_RECONCILE_REQUIRED, "reconcile required")
		return
	}
	input := make(chan *relaysyncv2.ClientFrame, 8)
	go s.readLoop(ctx, conn, input)
	ticker := time.NewTicker(s.options.HeartbeatInterval)
	defer ticker.Stop()
	flow := core.FlowControl{}
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-input:
			if frame == nil {
				return
			}
			switch {
			case frame.GetFlowControl() != nil:
				flow = core.FlowControl{MaxEvents: int(frame.GetFlowControl().MaxEvents), MaxBytes: int(frame.GetFlowControl().MaxBytes), Paused: frame.GetFlowControl().Pause}
				if err := syncer.SetFlowControl(flow); err != nil {
					_ = writeError(ctx, conn, relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_INVALID_MESSAGE, err.Error())
					return
				}
			case frame.GetAck() != nil:
				if err := syncer.Ack(ctx, cursorFromProto(frame.GetAck().Cursor), "ws-ack"); err != nil {
					_ = writeError(ctx, conn, relaysyncv2.ProtocolErrorCode_PROTOCOL_ERROR_INTERNAL, err.Error())
					return
				}
			case frame.GetHeartbeat() != nil:
				syncer.ObserveHeartbeat(time.Now())
			}
		case <-ticker.C:
			_, _ = syncer.Pump(ctx, 64)
			if batch, err := syncer.NextBatch(); err == nil {
				if err := writeFrame(ctx, conn, &relaysyncv2.ServerFrame{Body: &relaysyncv2.ServerFrame_Delta{Delta: deltaProto(batch)}}); err != nil {
					return
				}
			}
			heartbeat, due, err := syncer.HeartbeatDue(time.Now())
			if err != nil {
				return
			}
			if due {
				if err := writeFrame(ctx, conn, &relaysyncv2.ServerFrame{Body: &relaysyncv2.ServerFrame_Heartbeat{Heartbeat: &relaysyncv2.Heartbeat{Nonce: heartbeat.Nonce, SentAtUnixNano: heartbeat.SentAt.UnixNano()}}}); err != nil {
					return
				}
			}
		}
	}
}

func (s *Server) readLoop(ctx context.Context, conn *websocket.Conn, output chan<- *relaysyncv2.ClientFrame) {
	defer close(output)
	for {
		_, payload, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if len(payload) > s.options.MaxFrameBytes {
			return
		}
		frame := &relaysyncv2.ClientFrame{}
		if proto.Unmarshal(payload, frame) != nil {
			return
		}
		select {
		case output <- frame:
		case <-ctx.Done():
			return
		}
	}
}
func writeFrame(ctx context.Context, conn *websocket.Conn, frame proto.Message) error {
	payload, err := proto.Marshal(frame)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageBinary, payload)
}
func writeError(ctx context.Context, conn *websocket.Conn, code relaysyncv2.ProtocolErrorCode, message string) error {
	return writeFrame(ctx, conn, &relaysyncv2.ServerFrame{Body: &relaysyncv2.ServerFrame_Error{Error: &relaysyncv2.ProtocolError{Code: code, Message: message}}})
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
