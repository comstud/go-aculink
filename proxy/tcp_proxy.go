package proxy

import (
	"io"
	"log"
	"net"
	"os"
)

type TCPProxy struct {
	remoteAddr *net.TCPAddr
	Logger     *log.Logger
}

func (self *TCPProxy) Run(client *net.TCPConn) {
	server, err := net.DialTCP("tcp", nil, self.remoteAddr)
	if err != nil {
		return
	}

	defer server.Close()

	serverDone := make(done_chan, 1)
	clientDone := make(done_chan, 1)

	go func() {
		io.Copy(server, client)
		clientDone <- nil
	}()

	go func() {
		io.Copy(client, server)
		serverDone <- nil
	}()

	var allDone done_chan

	select {
	case <-clientDone:
		server.CloseRead()
		allDone = serverDone
	case <-serverDone:
		client.CloseRead()
		allDone = clientDone
	}

	<-allDone
}

func NewTCPProxy(dest_addr string) (*TCPProxy, error) {
	addr, err := net.ResolveTCPAddr("tcp", dest_addr)
	if err != nil {
		return nil, err
	}
	return &TCPProxy{
		remoteAddr: addr,
		Logger: log.New(
			os.Stdout,
			"",
			log.LstdFlags|log.Lmicroseconds,
		),
	}, nil
}
