package proxy

import (
	"net"
	"net/http"
)

type done_chan chan interface{}

func dial(network string, addr string) (conn net.Conn, err error) {
	conn, err = net.Dial(network, addr)
	if err != nil {
		return
	}
	if tcp_conn, ok := conn.(*net.TCPConn); ok {
		tcp_conn.SetKeepAlive(true)
	}
	return
}

func copyHeaders(dst http.Header, src http.Header) {
	for k, _ := range dst {
		dst.Del(k)
	}
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}
