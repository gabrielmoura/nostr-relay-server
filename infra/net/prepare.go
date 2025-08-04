package net

import (
	"net"
)

func PrepareListen(addr string) (net.Listener, error) {
	//if srv.shuttingDown() {
	//	return nil,http.ErrServerClosed
	//}
	//addr := srv.Addr
	//if addr == "" {
	//	addr = ":http"
	//}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return ln, nil
}
