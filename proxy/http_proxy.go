package proxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/comstud/go-aculink/aculink"
)

type AculinkProxy struct {
	Transport    *http.Transport
	RedirectPort int
	Logger       *log.Logger
	Db           *aculink.DB
}

func (self *AculinkProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	self.Logger.Printf("Received: %s %s", r.Method, r.URL)
	if r.Host == "www.acu-link.com" {
		if r.Method == "POST" {
			buf, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading body", http.StatusInternalServerError)
				return
			}
			r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
			go self.HandleSensorData(buf)
		}
		r.Host = "acu-link.com"
		self.ProxyRequest(w, r)
		return
	}
	// Try redirect to server running on another port
	r.URL.Scheme = "http"
	r.URL.Host = r.Host + fmt.Sprintf(":%d", self.RedirectPort)
	http.Redirect(w, r, r.URL.String(), http.StatusFound)
}

func (self *AculinkProxy) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	r.URL.Scheme = "http"
	r.URL.Host = r.Host

	resp, err := self.Transport.RoundTrip(r)
	if err != nil {
		http.Error(w, "Error handling request", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (self *AculinkProxy) HandleSensorData(data []byte) {
	s := string(data)
	d, err := aculink.NewData(s)
	if err != nil {
		self.Logger.Printf("Error parsing data '%s': %s", s, err.Error())
		return
	}
	self.Logger.Printf("DATA: %s", d.JSONString())
	if self.Db != nil {
		exists, err := self.Db.UUIDExists(d.UUID.String())
		if err != nil {
			self.Logger.Printf("Error adding to DB: %s", err.Error())
			return
		}

		if exists {
			self.Logger.Printf(
				"Error adding to DB: uuid %s already exists",
				d.UUID,
			)
			return
		}

		err = self.Db.InsertData(d)
		if err != nil {
			self.Logger.Printf("Error adding to DB: %s", err.Error())
			return
		}
	}
}

func NewAculinkProxy() *AculinkProxy {
	return &AculinkProxy{
		Transport: &http.Transport{
			Dial: dial,
		},
		RedirectPort: 8080,
		Logger: log.New(
			os.Stdout,
			"",
			log.LstdFlags|log.Lmicroseconds,
		),
	}
}
