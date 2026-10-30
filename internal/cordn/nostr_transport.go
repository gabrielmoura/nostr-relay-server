package cordn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/thomas-vilte/mls-go/credentials"
	"github.com/thomas-vilte/mls-go/keypackages"
)

const contextVMKind = 25910

// NostrCoordinator is the plaintext ContextVM transport for Cordn. It stores
// only opaque MLS values; neither this transport nor the relay decrypts MLS
// group content. Encrypted ContextVM envelopes are intentionally not accepted
// here because the relay does not yet implement their NIP-59 unwrap boundary.
type NostrCoordinator struct {
	service                *Service
	privKey                string
	pubKey                 string
	sem                    chan struct{}
	maxMessageBytes        int
	maxPendingJoinRequests int
}

func NewNostrCoordinator(service *Service, privKey, pubKey string, maxSubscriptions, maxMessageBytes, maxPendingJoinRequests int) (*NostrCoordinator, error) {
	if service == nil {
		return nil, errors.New("cordn: service is required")
	}
	if strings.TrimSpace(privKey) == "" || strings.TrimSpace(pubKey) == "" {
		return nil, errors.New("cordn: relay signing keys are required")
	}
	if maxSubscriptions <= 0 {
		return nil, errors.New("cordn: max concurrent subscriptions must be positive")
	}
	if maxMessageBytes <= 0 || maxPendingJoinRequests <= 0 {
		return nil, errors.New("cordn: message and pending join request limits must be positive")
	}
	return &NostrCoordinator{service: service, privKey: privKey, pubKey: pubKey, sem: make(chan struct{}, maxSubscriptions), maxMessageBytes: maxMessageBytes, maxPendingJoinRequests: maxPendingJoinRequests}, nil
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Meta      struct {
		ProgressToken string `json:"progressToken"`
	} `json:"_meta"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Handle handles a signed ContextVM request addressed to this relay. send must
// be safe to call from a goroutine and must stop delivery when ctx is canceled.
// It returns false for unrelated Nostr events so normal relay handling remains
// unchanged.
func (c *NostrCoordinator) Handle(ctx context.Context, request *nostr.Event, send func(nostr.Event)) bool {
	if request == nil || request.Kind != contextVMKind || !hasPTarget(request.Tags, c.pubKey) {
		return false
	}

	var rpc rpcRequest
	if err := json.Unmarshal([]byte(request.Content), &rpc); err != nil {
		c.sendError(request, send, nil, -32700, "invalid JSON-RPC request")
		return true
	}
	if rpc.JSONRPC != "2.0" {
		c.sendError(request, send, rpc.ID, -32600, "expected JSON-RPC 2.0 request")
		return true
	}
	// MCP clients initialize before their first tools/call. Notifications do not
	// have an id and deliberately receive no response.
	if strings.HasPrefix(rpc.Method, "notifications/") {
		return true
	}
	if rpc.Method == "initialize" {
		c.sendResponse(request, send, rpcResponse{JSONRPC: "2.0", ID: rpc.ID, Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "nrserver-cordn", "version": "1.0.0"},
		}})
		return true
	}
	if rpc.Method == "tools/list" {
		c.sendResponse(request, send, rpcResponse{JSONRPC: "2.0", ID: rpc.ID, Result: map[string]any{"tools": cordnTools()}})
		return true
	}
	if len(rpc.ID) == 0 || rpc.Method != "tools/call" {
		c.sendError(request, send, rpc.ID, -32600, "expected JSON-RPC 2.0 tools/call request")
		return true
	}
	var call toolCallParams
	if err := json.Unmarshal(rpc.Params, &call); err != nil || call.Name == "" {
		c.sendError(request, send, rpc.ID, -32602, "invalid tools/call params")
		return true
	}

	if call.Name == "msg_sub_many" {
		if call.Meta.ProgressToken == "" {
			c.sendError(request, send, rpc.ID, -32602, "msg_sub_many requires _meta.progressToken")
			return true
		}
		groups, err := decodeGroups(call.Arguments)
		if err != nil {
			c.sendError(request, send, rpc.ID, -32602, err.Error())
			return true
		}
		done, err := c.subscribeMany(ctx, request, groups, call.Meta.ProgressToken, send)
		if err != nil {
			c.sendError(request, send, rpc.ID, -32000, err.Error())
			return true
		}
		go func() {
			<-done
			c.sendResponse(request, send, rpcResponse{JSONRPC: "2.0", ID: rpc.ID, Result: map[string]any{"content": []any{}, "structuredContent": map[string]any{"subscribed": true, "groups": groupIDs(groups)}}})
		}()
		return true
	}

	result, err := c.call(ctx, request, call, send)
	if err != nil {
		c.sendError(request, send, rpc.ID, -32000, err.Error())
		return true
	}
	c.sendResponse(request, send, rpcResponse{JSONRPC: "2.0", ID: rpc.ID, Result: map[string]any{"content": []any{}, "structuredContent": result}})
	return true
}

func (c *NostrCoordinator) CleanupExpired(ctx context.Context, before time.Time) error {
	return c.service.CleanupExpired(ctx, before)
}

func (c *NostrCoordinator) call(ctx context.Context, request *nostr.Event, call toolCallParams, send func(nostr.Event)) (any, error) {
	switch call.Name {
	case "kp_publish":
		var input struct {
			Reference string `json:"kp_ref"`
			Payload   string `json:"kp_64"`
		}
		if err := decodeArguments(call.Arguments, &input); err != nil || input.Reference == "" {
			return nil, errors.New("invalid kp_publish arguments")
		}
		payload, err := decodeBase64(input.Payload)
		if err != nil {
			return nil, errors.New("invalid kp_64")
		}
		keyPackage, err := keypackages.UnmarshalKeyPackage(payload)
		if err != nil || keyPackage.Validate() != nil || keyPackage.Verify(keyPackage.CipherSuite) != nil || keyPackage.LeafNode == nil || keyPackage.LeafNode.Credential == nil {
			return nil, errors.New("invalid MLS key package")
		}
		if keyPackage.LeafNode.Credential.CredentialType != credentials.BasicCredential {
			return nil, errors.New("only BasicCredential key packages are supported")
		}
		if keyPackage.LeafNode.Credential.IdentityString() != request.PubKey {
			return nil, errors.New("key package credential identity does not match request signer")
		}
		publication, err := json.Marshal(request)
		if err != nil {
			return nil, fmt.Errorf("marshal publication event: %w", err)
		}
		value := KeyPackage{Reference: input.Reference, StablePubkey: request.PubKey, Payload: payload, PublicationJSON: publication, LastResort: isCordnLastResort(keyPackage), PublishedAt: time.Now().UTC()}
		if err := c.service.PublishKeyPackage(ctx, value); err != nil {
			return nil, err
		}
		return map[string]any{"kp_ref": input.Reference, "last_resort": value.LastResort, "at": millis(value.PublishedAt)}, nil

	case "kp_list":
		values, err := c.service.ListKeyPackages(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]any, 0, len(values))
		for _, value := range values {
			items = append(items, keyPackageSummary(value))
		}
		return map[string]any{"keyPackages": items}, nil

	case "kp_take":
		var input struct {
			ID string `json:"id"`
		}
		if err := decodeArguments(call.Arguments, &input); err != nil || input.ID == "" {
			return nil, errors.New("invalid kp_take arguments")
		}
		value, found, err := c.service.TakeKeyPackage(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		if !found {
			return map[string]any{"keyPackage": nil}, nil
		}
		var event any
		if err := json.Unmarshal(value.PublicationJSON, &event); err != nil {
			return nil, fmt.Errorf("decode stored publication event: %w", err)
		}
		return map[string]any{"keyPackage": map[string]any{"pk": value.StablePubkey, "kp_ref": value.Reference, "last_resort": value.LastResort, "at": millis(value.PublishedAt), "event": event}}, nil

	case "kp_remove":
		var input struct {
			References []string `json:"kp_refs"`
		}
		if err := decodeArguments(call.Arguments, &input); err != nil || len(input.References) == 0 {
			return nil, errors.New("invalid kp_remove arguments")
		}
		removed := make([]string, 0, len(input.References))
		for _, reference := range input.References {
			ok, err := c.service.RemoveKeyPackage(ctx, reference, request.PubKey)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("unknown or unauthorized key package ref: %s", reference)
			}
			removed = append(removed, reference)
		}
		return map[string]any{"kp_refs": removed}, nil

	case "welcome_store":
		var input struct {
			Target    string `json:"target_pk"`
			Reference string `json:"kp_ref"`
			Payload   string `json:"welcome_64"`
			After     *int64 `json:"after"`
		}
		if err := decodeArguments(call.Arguments, &input); err != nil || input.Target == "" || input.Reference == "" {
			return nil, errors.New("invalid welcome_store arguments")
		}
		payload, err := decodeBase64(input.Payload)
		if err != nil {
			return nil, errors.New("invalid welcome_64")
		}
		value := Welcome{TargetPubkey: input.Target, KeyPackageRef: input.Reference, Payload: payload, JoinAfterCursor: input.After, CreatedAt: nowMillis()}
		if err := c.service.StoreWelcome(ctx, value); err != nil {
			return nil, err
		}
		return map[string]any{"at": millis(value.CreatedAt)}, nil

	case "welcome_take":
		var input struct {
			Consumed []struct {
				Reference string `json:"kp_ref"`
				At        int64  `json:"at"`
			} `json:"consumed"`
		}
		if err := decodeArguments(call.Arguments, &input); err != nil {
			return nil, errors.New("invalid welcome_take arguments")
		}
		consumed := make([]Welcome, 0, len(input.Consumed))
		for _, item := range input.Consumed {
			consumed = append(consumed, Welcome{KeyPackageRef: item.Reference, CreatedAt: time.UnixMilli(item.At).UTC()})
		}
		values, err := c.service.TakeWelcomes(ctx, request.PubKey, consumed)
		if err != nil {
			return nil, err
		}
		items := make([]any, 0, len(values))
		for _, value := range values {
			items = append(items, welcomeWire(value))
		}
		return map[string]any{"welcomes": items}, nil

	case "join_request_store":
		var input struct {
			GroupID   string `json:"gid"`
			Reference string `json:"kp_ref"`
		}
		if err := decodeArguments(call.Arguments, &input); err != nil || input.GroupID == "" || input.Reference == "" {
			return nil, errors.New("invalid join_request_store arguments")
		}
		keyPackage, found, err := c.service.GetKeyPackage(ctx, input.Reference)
		if err != nil {
			return nil, err
		}
		if !found || keyPackage.StablePubkey != request.PubKey {
			return nil, errors.New("unknown or unauthorized key package ref")
		}
		pending, err := c.service.ListJoinRequests(ctx, input.GroupID)
		if err != nil {
			return nil, err
		}
		if len(pending) >= c.maxPendingJoinRequests {
			return nil, errors.New("Cordn pending join request limit reached")
		}
		value := JoinRequest{GroupID: input.GroupID, RequesterPubkey: request.PubKey, KeyPackageRef: input.Reference, CreatedAt: nowMillis()}
		if err := c.service.StoreJoinRequest(ctx, value); err != nil {
			return nil, err
		}
		return map[string]any{"at": millis(value.CreatedAt)}, nil

	case "join_request_take_many":
		var input struct {
			Groups []struct {
				GroupID string `json:"gid"`
			} `json:"groups"`
			Consumed []struct {
				GroupID string `json:"gid"`
				Pubkey  string `json:"pk"`
				At      int64  `json:"at"`
			} `json:"consumed"`
		}
		if err := decodeArguments(call.Arguments, &input); err != nil || len(input.Groups) == 0 {
			return nil, errors.New("invalid join_request_take_many arguments")
		}
		items := []any{}
		for _, group := range input.Groups {
			consumed := make([]JoinRequest, 0)
			for _, item := range input.Consumed {
				if item.GroupID == group.GroupID {
					consumed = append(consumed, JoinRequest{GroupID: item.GroupID, RequesterPubkey: item.Pubkey, CreatedAt: time.UnixMilli(item.At).UTC()})
				}
			}
			values, err := c.service.TakeJoinRequests(ctx, group.GroupID, consumed)
			if err != nil {
				return nil, err
			}
			for _, value := range values {
				items = append(items, map[string]any{"gid": value.GroupID, "pk": value.RequesterPubkey, "kp_ref": value.KeyPackageRef, "at": millis(value.CreatedAt)})
			}
		}
		return map[string]any{"requests": items}, nil

	case "msg_post":
		var input struct {
			GroupID string `json:"gid"`
			Payload string `json:"msg_64"`
		}
		if err := decodeArguments(call.Arguments, &input); err != nil || input.GroupID == "" {
			return nil, errors.New("invalid msg_post arguments")
		}
		payload, err := decodeBase64(input.Payload)
		if err != nil {
			return nil, errors.New("invalid msg_64")
		}
		if len(payload) > c.maxMessageBytes {
			return nil, errors.New("Cordn message exceeds configured limit")
		}
		value, err := c.service.PublishGroupMessage(ctx, input.GroupID, payload)
		if err != nil {
			return nil, err
		}
		return map[string]any{"cursor": value.Cursor, "gid": value.GroupID, "at": millis(value.CreatedAt)}, nil

	case "msg_fetch_many":
		groups, err := decodeGroups(call.Arguments)
		if err != nil {
			return nil, err
		}
		return c.fetchMany(ctx, groups)

	default:
		return nil, fmt.Errorf("unsupported Cordn tool: %s", call.Name)
	}
}

type groupCursor struct {
	GroupID string `json:"gid"`
	After   *int64 `json:"after"`
}

func decodeGroups(raw json.RawMessage) ([]groupCursor, error) {
	var input struct {
		Groups []groupCursor `json:"groups"`
	}
	if err := decodeArguments(raw, &input); err != nil || len(input.Groups) == 0 {
		return nil, errors.New("invalid group arguments")
	}
	for _, group := range input.Groups {
		if group.GroupID == "" || (group.After != nil && *group.After < 0) {
			return nil, errors.New("invalid group arguments")
		}
	}
	return input.Groups, nil
}

func (c *NostrCoordinator) fetchMany(ctx context.Context, groups []groupCursor) (any, error) {
	items := []any{}
	for _, group := range groups {
		after := int64(0)
		if group.After != nil {
			after = *group.After
		}
		values, err := c.service.FetchGroupMessages(ctx, group.GroupID, after)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			items = append(items, messageWire(value))
		}
	}
	return map[string]any{"messages": items}, nil
}

func (c *NostrCoordinator) subscribeMany(ctx context.Context, request *nostr.Event, groups []groupCursor, progressToken string, send func(nostr.Event)) (<-chan struct{}, error) {
	type preparedSubscription struct {
		backlog []GroupMessage
		live    <-chan GroupMessage
		cancel  func()
	}
	prepared := make([]preparedSubscription, 0, len(groups))
	for _, group := range groups {
		select {
		case c.sem <- struct{}{}:
		default:
			for _, value := range prepared {
				value.cancel()
				<-c.sem
			}
			return nil, errors.New("Cordn subscription limit reached")
		}
		after := int64(0)
		if group.After != nil {
			after = *group.After
		}
		backlog, live, cancel, err := c.service.SubscribeGroupMessages(ctx, group.GroupID, after)
		if err != nil {
			for _, value := range prepared {
				value.cancel()
				<-c.sem
			}
			<-c.sem
			return nil, err
		}
		prepared = append(prepared, preparedSubscription{backlog: backlog, live: live, cancel: cancel})
	}
	writer := &cordnStreamWriter{coordinator: c, request: request, token: progressToken, send: send}
	writer.start()
	done := make(chan struct{})
	var wait sync.WaitGroup
	for _, value := range prepared {
		wait.Add(1)
		go func(value preparedSubscription) {
			defer wait.Done()
			c.forwardGroup(ctx, value.backlog, value.live, value.cancel, writer)
		}(value)
	}
	go func() { wait.Wait(); writer.close(); close(done) }()
	return done, nil
}

func (c *NostrCoordinator) forwardGroup(ctx context.Context, backlog []GroupMessage, live <-chan GroupMessage, cancel func(), writer *cordnStreamWriter) {
	defer func() { cancel(); <-c.sem }()
	last := int64(0)
	for _, value := range backlog {
		if value.Cursor > last {
			last = value.Cursor
			writer.write(messageWire(value))
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case value, ok := <-live:
			if !ok {
				return
			}
			if value.Cursor <= last {
				continue
			}
			last = value.Cursor
			writer.write(messageWire(value))
		}
	}
}

type cordnStreamWriter struct {
	coordinator *NostrCoordinator
	request     *nostr.Event
	token       string
	send        func(nostr.Event)
	mu          sync.Mutex
	progress    int
	chunk       int
}

func (w *cordnStreamWriter) start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.publishLocked(map[string]any{"type": "open-stream", "frameType": "start"})
}
func (w *cordnStreamWriter) write(value any) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	chunk := w.chunk
	w.chunk++
	w.publishLocked(map[string]any{"type": "open-stream", "frameType": "chunk", "chunkIndex": chunk, "data": string(encoded)})
}
func (w *cordnStreamWriter) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	last := w.chunk - 1
	frame := map[string]any{"type": "open-stream", "frameType": "close"}
	if last >= 0 {
		frame["lastChunkIndex"] = last
	}
	w.publishLocked(frame)
}
func (w *cordnStreamWriter) publish(frame map[string]any) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.publishLocked(frame)
}
func (w *cordnStreamWriter) publishLocked(frame map[string]any) {
	progress := w.progress
	w.progress++
	w.coordinator.sendNotification(w.request, w.send, "notifications/progress", map[string]any{"progressToken": w.token, "progress": progress, "cvm": frame})
}

func (c *NostrCoordinator) sendNotification(request *nostr.Event, send func(nostr.Event), method string, params any) {
	c.sendRaw(request, send, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (c *NostrCoordinator) sendError(request *nostr.Event, send func(nostr.Event), id json.RawMessage, code int, message string) {
	c.sendResponse(request, send, rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}

func (c *NostrCoordinator) sendResponse(request *nostr.Event, send func(nostr.Event), response rpcResponse) {
	c.sendRaw(request, send, response)
}

func (c *NostrCoordinator) sendRaw(request *nostr.Event, send func(nostr.Event), response any) {
	content, err := json.Marshal(response)
	if err != nil {
		return
	}
	event := nostr.Event{Kind: contextVMKind, PubKey: c.pubKey, CreatedAt: nostr.Now(), Content: string(content), Tags: nostr.Tags{{"p", request.PubKey}, {"e", request.ID}}}
	if err := event.Sign(c.privKey); err == nil {
		send(event)
	}
}

func decodeArguments(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	return json.Unmarshal(raw, target)
}

func decodeBase64(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("empty base64")
	}
	return base64.StdEncoding.DecodeString(value)
}

func hasPTarget(tags nostr.Tags, pubkey string) bool {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == "p" && tag[1] == pubkey {
			return true
		}
	}
	return false
}
func millis(value time.Time) int64 { return value.UnixMilli() }
func nowMillis() time.Time         { return time.Now().UTC().Truncate(time.Millisecond) }
func keyPackageSummary(value KeyPackage) any {
	return map[string]any{"pk": value.StablePubkey, "kp_ref": value.Reference, "last_resort": value.LastResort, "at": millis(value.PublishedAt)}
}
func welcomeWire(value Welcome) any {
	output := map[string]any{"kp_ref": value.KeyPackageRef, "welcome_64": base64.StdEncoding.EncodeToString(value.Payload), "at": millis(value.CreatedAt)}
	if value.JoinAfterCursor != nil {
		output["after"] = *value.JoinAfterCursor
	}
	return output
}
func messageWire(value GroupMessage) any {
	return map[string]any{"cursor": value.Cursor, "gid": value.GroupID, "msg_64": base64.StdEncoding.EncodeToString(value.Payload), "at": millis(value.CreatedAt)}
}

func cordnTools() []any {
	names := []string{"kp_publish", "kp_list", "kp_take", "kp_remove", "welcome_take", "welcome_store", "join_request_store", "join_request_take_many", "msg_post", "msg_fetch_many", "msg_sub_many"}
	tools := make([]any, 0, len(names))
	for _, name := range names {
		tools = append(tools, map[string]any{"name": name, "inputSchema": map[string]any{"type": "object"}})
	}
	return tools
}

func groupIDs(groups []groupCursor) []string {
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.GroupID)
	}
	return ids
}

// Cordn encodes its reusable last-resort marker in the MLS application-data
// dictionary (extension 0x0006, component 0x0004 with an empty value). It is
// deliberately distinct from the standard MLS LastResort extension.
func isCordnLastResort(keyPackage *keypackages.KeyPackage) bool {
	for _, extension := range keyPackage.Extensions {
		if uint16(extension.Type) != 0x0006 {
			continue
		}
		dictionary, next, ok := readCordnVarBytes(extension.Data, 0)
		if !ok || next != len(extension.Data) {
			return false
		}
		for offset := 0; offset < len(dictionary); {
			if offset+2 > len(dictionary) {
				return false
			}
			componentID := int(dictionary[offset])<<8 | int(dictionary[offset+1])
			component, nextOffset, ok := readCordnVarBytes(dictionary, offset+2)
			if !ok {
				return false
			}
			if componentID == 0x0004 && len(component) == 0 {
				return true
			}
			offset = nextOffset
		}
	}
	return false
}

func readCordnVarBytes(value []byte, offset int) ([]byte, int, bool) {
	if offset >= len(value) {
		return nil, offset, false
	}
	lengthSize := 1 << ((value[offset] & 0xc0) >> 6)
	if offset+lengthSize > len(value) {
		return nil, offset, false
	}
	length := int(value[offset] & 0x3f)
	for index := 1; index < lengthSize; index++ {
		length = length<<8 | int(value[offset+index])
	}
	start := offset + lengthSize
	end := start + length
	if end > len(value) {
		return nil, offset, false
	}
	return value[start:end], end, true
}
