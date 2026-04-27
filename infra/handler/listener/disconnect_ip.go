package listener

import "github.com/gabrielmoura/nostr-relay-server/internal/dto"

func DisconnectByIP(ip string) int {
	wsToIDMutex.RLock()
	targets := make([]*dto.WsServer, 0)
	for ws := range wsToID {
		if ws != nil && ws.RemoteIP == ip {
			targets = append(targets, ws)
		}
	}
	wsToIDMutex.RUnlock()

	disconnected := 0
	for _, ws := range targets {
		if disconnectSocket(ws) {
			disconnected++
		}
	}
	return disconnected
}

func disconnectSocket(target *dto.WsServer) bool {
	if target == nil || target.Conn == nil {
		return false
	}
	RemoveListener(target)
	_ = target.Conn.Close()
	return true
}
