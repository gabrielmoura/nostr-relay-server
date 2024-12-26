package net

import (
	"github.com/gabrielmoura/nostr-relay-server/internal/dto"
	"github.com/goccy/go-json"
	"net"
	"net/http"
)

func PrepareListen(srv *http.Server) (net.Listener, error) {
	//if srv.shuttingDown() {
	//	return nil,http.ErrServerClosed
	//}
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return ln, nil
}

func GetRealIp(ws *dto.WsServer) string {
	ip := ws.Conn.RemoteAddr().String()
	if realIP := ws.Request.Header.Get("X-Forwarded-For"); realIP != "" {
		ip = realIP // possible to be multiple comma separated
	} else if realIP := ws.Request.Header.Get("X-Real-Ip"); realIP != "" {
		ip = realIP
	}
	return ip
}

func JsonResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
