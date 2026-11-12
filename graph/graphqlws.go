package graph

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// The admin panel keeps queries and mutations on the authenticated HTTP POST
// endpoint. This handler implements only the graphql-transport-ws subset needed
// for subscriptions, which avoids adapting gqlgen's gorilla upgrader to Fiber.
const (
	wsConnectionInit  = "connection_init"
	wsConnectionAck   = "connection_ack"
	wsSubscribe       = "subscribe"
	wsNext            = "next"
	wsError           = "error"
	wsComplete        = "complete"
	wsPing            = "ping"
	wsPong            = "pong"
	wsConnectionError = "connection_error"
	wsTerminate       = "connection_terminate"

	// 4401 is the graphql-ws convention for unauthorized connections.
	wsCloseUnauthorized = 4401
	wsCloseBadRequest   = 4400
	wsCloseNormal       = 1000
)

// WebSocketHandler returns a Fiber handler that upgrades /admin/graphql to
// graphql-transport-ws. token is the admin API token; an empty token disables
// authentication, matching AdminTokenMiddleware semantics.
func WebSocketHandler(token string) fiber.Handler {
	exec := newSubscriptionExecutor(&Resolver{})
	expected := strings.TrimSpace(token)

	return websocket.New(func(conn *websocket.Conn) {
		newWSConnection(conn, exec, expected).run()
	}, websocket.Config{
		Subprotocols: []string{"graphql-transport-ws"},
	})
}

func newSubscriptionExecutor(resolver *Resolver) *executor.Executor {
	exec := executor.New(NewExecutableSchema(Config{Resolvers: resolver}))
	exec.SetErrorPresenter(graphQLErrorPresenter)
	exec.SetRecoverFunc(graphQLRecover)
	return exec
}

type wsClientMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type wsServerMessage struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type wsConnection struct {
	conn     *websocket.Conn
	exec     *executor.Executor
	expected string

	ctx    context.Context
	cancel context.CancelFunc

	writeMu sync.Mutex
	mu      sync.Mutex
	active  map[string]context.CancelFunc
	closed  bool
}

func newWSConnection(conn *websocket.Conn, exec *executor.Executor, expected string) *wsConnection {
	ctx, cancel := context.WithCancel(context.Background())
	return &wsConnection{
		conn:     conn,
		exec:     exec,
		expected: expected,
		ctx:      ctx,
		cancel:   cancel,
		active:   make(map[string]context.CancelFunc),
	}
}

func (c *wsConnection) run() {
	defer func() {
		c.cancel()
		c.closeActive()
		_ = c.conn.Close()
	}()

	initialized := false
	for {
		var message wsClientMessage
		if err := c.conn.ReadJSON(&message); err != nil {
			return
		}

		switch message.Type {
		case wsConnectionInit:
			if initialized {
				c.sendConnectionError("too many initialization requests")
				c.close(wsCloseBadRequest, "too many initialization requests")
				return
			}
			if !c.authenticate(message.Payload) {
				c.sendConnectionError("invalid admin token")
				c.close(wsCloseUnauthorized, "invalid admin token")
				return
			}
			initialized = true
			c.write(wsServerMessage{Type: wsConnectionAck})
		case wsPing:
			c.write(wsServerMessage{Type: wsPong, Payload: rawPayload(message.Payload)})
		case wsPong:
			// Keepalive acknowledgement; nothing to do.
		case wsSubscribe:
			if !initialized {
				c.sendConnectionError("connection is not initialised")
				c.close(wsCloseBadRequest, "connection is not initialised")
				return
			}
			if !c.startSubscription(message.ID, message.Payload) {
				return
			}
		case wsComplete, wsTerminate:
			if message.ID != "" {
				c.stopSubscription(message.ID)
			}
			if message.Type == wsTerminate {
				c.close(wsCloseNormal, "terminated")
				return
			}
		default:
			c.sendConnectionError("unexpected message type %q", message.Type)
			c.close(wsCloseBadRequest, "unexpected message")
			return
		}
	}
}

func (c *wsConnection) authenticate(payload json.RawMessage) bool {
	if c.expected == "" {
		return true
	}
	var init struct {
		Token         string `json:"token"`
		Authorization string `json:"authorization"`
	}
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &init)
	}
	candidate := strings.TrimSpace(init.Token)
	if candidate == "" {
		candidate = strings.TrimSpace(strings.TrimPrefix(init.Authorization, "Bearer "))
	}
	if candidate == "" {
		if header, ok := c.conn.Locals("admin-token").(string); ok {
			candidate = strings.TrimSpace(header)
		}
	}
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(c.expected)) == 1
}

func (c *wsConnection) startSubscription(id string, payload json.RawMessage) bool {
	if strings.TrimSpace(id) == "" {
		c.sendConnectionError("subscription id is required")
		c.close(wsCloseBadRequest, "missing subscription id")
		return false
	}

	params := &graphql.RawParams{}
	if err := json.Unmarshal(payload, params); err != nil {
		c.sendError(id, &gqlerror.Error{Message: "invalid subscription payload"})
		c.write(wsServerMessage{ID: id, Type: wsComplete})
		return true
	}
	params.ReadTime = graphql.TraceTiming{Start: graphql.Now(), End: graphql.Now()}
	params.Headers = wsHeaders(c.conn)

	ctx := graphql.StartOperationTrace(c.ctx)
	opCtx, errs := c.exec.CreateOperationContext(ctx, params)
	if len(errs) > 0 {
		c.sendError(id, errs...)
		c.write(wsServerMessage{ID: id, Type: wsComplete})
		return true
	}
	if opCtx.Operation == nil || opCtx.Operation.Operation != ast.Subscription {
		c.sendError(id, &gqlerror.Error{Message: "only subscriptions are supported over the WebSocket transport; send queries and mutations to POST /admin/graphql"})
		c.write(wsServerMessage{ID: id, Type: wsComplete})
		return true
	}

	c.mu.Lock()
	if _, exists := c.active[id]; exists {
		c.mu.Unlock()
		c.sendError(id, &gqlerror.Error{Message: "subscription id is already in use"})
		c.write(wsServerMessage{ID: id, Type: wsComplete})
		return true
	}
	subCtx, cancel := context.WithCancel(ctx)
	c.active[id] = cancel
	c.mu.Unlock()

	go c.runSubscription(id, subCtx, cancel, opCtx)
	return true
}

func (c *wsConnection) runSubscription(id string, ctx context.Context, cancel context.CancelFunc, opCtx *graphql.OperationContext) {
	defer func() {
		if r := recover(); r != nil {
			err := opCtx.RecoverFunc(ctx, r)
			if gqlErr, ok := err.(*gqlerror.Error); ok {
				c.sendError(id, gqlErr)
			} else if err != nil {
				c.sendError(id, &gqlerror.Error{Message: err.Error()})
			}
		}
		c.mu.Lock()
		delete(c.active, id)
		c.mu.Unlock()
		cancel()
		c.write(wsServerMessage{ID: id, Type: wsComplete})
	}()

	responses, respCtx := c.exec.DispatchOperation(ctx, opCtx)
	for {
		response := responses(respCtx)
		if response == nil {
			return
		}
		if ctx.Err() != nil {
			return
		}
		c.write(wsServerMessage{ID: id, Type: wsNext, Payload: response})
	}
}

func (c *wsConnection) stopSubscription(id string) {
	c.mu.Lock()
	cancel := c.active[id]
	delete(c.active, id)
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *wsConnection) closeActive() {
	c.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(c.active))
	for _, cancel := range c.active {
		cancels = append(cancels, cancel)
	}
	c.active = make(map[string]context.CancelFunc)
	c.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (c *wsConnection) write(message wsServerMessage) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.WriteJSON(message)
}

func (c *wsConnection) sendError(id string, errs ...*gqlerror.Error) {
	c.write(wsServerMessage{ID: id, Type: wsError, Payload: gqlerror.List(errs)})
}

func (c *wsConnection) sendConnectionError(format string, args ...any) {
	c.write(wsServerMessage{Type: wsConnectionError, Payload: gqlerror.Errorf(format, args...)})
}

func (c *wsConnection) close(code int, reason string) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	deadline := time.Now().Add(time.Second)
	_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), deadline)
}

func wsHeaders(conn *websocket.Conn) http.Header {
	headers := http.Header{}
	for _, key := range []string{"User-Agent", "Origin", "X-Forwarded-For", "X-Real-Ip"} {
		if value := strings.TrimSpace(conn.Headers(key)); value != "" {
			headers.Set(key, value)
		}
	}
	return headers
}

func rawPayload(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 {
		return json.RawMessage("null")
	}
	return payload
}
