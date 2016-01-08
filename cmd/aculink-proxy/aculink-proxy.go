package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/comstud/go-aculink/aculink"
	"github.com/comstud/go-aculink/proxy"
)

const DEFAULT_LOGFILE = "/var/log/aculink-proxy.log"
const DEFAULT_DB_DSN = "aculink:aculink@/aculink"

func getLogger() *log.Logger {
	logfile := os.Getenv("ACULINK_PROXY_LOGFILE")
	if logfile == "" {
		logfile = DEFAULT_LOGFILE
	}

	f, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return log.New(
		f,
		"",
		log.LstdFlags|log.Lmicroseconds,
	)
}

func getDB() *aculink.DB {
	db_dsn := os.Getenv("ACULINK_PROXY_DB_DSN")
	if db_dsn == "" {
		db_dsn = DEFAULT_DB_DSN
	}

	db, err := aculink.OpenDB(db_dsn)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func main() {
	db := getDB()
	logger := getLogger()

	http_proxy := proxy.NewAculinkProxy()
	http_proxy.Logger = logger
	http_proxy.Db = db

	https_proxy, err := proxy.NewTCPProxy("acu-link.com:443")
	if err != nil {
		log.Fatal(err)
	}
	https_proxy.Logger = logger

	http_listener, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal(err)
	}

	https_listener, err := net.Listen("tcp", ":443")
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan error, 1)

	go func() {
		done <- http.Serve(http_listener, http_proxy)
	}()

	go func() {
		var tempDelay = 2 * time.Millisecond

		listener := https_listener.(*net.TCPListener)
		defer listener.Close()

		for {
			tcp_conn, err := listener.AcceptTCP()
			if err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
					tempDelay *= 2
					if max := 1 * time.Second; tempDelay > max {
						tempDelay = max
					}
					time.Sleep(tempDelay)
					continue
				}
				done <- err
			}
			tcp_conn.SetKeepAlive(true)
			go func() {
				defer tcp_conn.Close()
				log.Printf(
					"Proxying HTTPs for client %s",
					tcp_conn.RemoteAddr(),
				)
				https_proxy.Run(tcp_conn)
			}()
		}
	}()

	log.Fatal(<-done)
}
