package net

import (
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
