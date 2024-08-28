package crux

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/erixzone/crux/pkg/walrus"
	"github.com/erixzone/crypto/pkg/tls"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PromHandler : struct for prometheus HttpHandler and associated "quit" channel
type PromHandler struct {
	sync.Mutex
	port        int
	waitForQuit bool
	runComplete bool
	quitChan    chan bool
	prom        http.Handler
	qcount      int
	ptotal      int
	ppost       int
}

// NewPromHandler : rhubarb
func NewPromHandler(port int, wfq bool) *PromHandler {
	h := &PromHandler{
		port:        port,
		waitForQuit: wfq,
		quitChan:    make(chan bool, 1),
		prom:        promhttp.Handler(),
	}
	http.Handle("/metrics", h)
	http.Handle("/quit", h)
	return h
}

// WaitFinish : wait for last scrapes or a "quit" message
func (h *PromHandler) WaitFinish() {
	h.Lock()
	wuzComplete := h.runComplete
	h.runComplete = true
	h.Unlock()
	if wuzComplete { // won't happen with crux.Exit
		return
	}
	if len(h.quitChan) > 0 {
		// pass
	} else if h.waitForQuit {
		fmt.Printf("waiting on quit channel...\n")
	} else if h.ptotal > 0 {
		fmt.Printf("waiting for final scrapes...\n")
	} else {
		fmt.Printf("no scrapers, no waiting.\n")
		return
	}
	<-h.quitChan
	fmt.Printf("quit message received.\n")
}

// ServeHTTP : handler for prometheus and associated "quit" channel
func (h *PromHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Lock()
	defer h.Unlock()
	switch r.URL.Path {
	case "/quit":
		h.qcount++
		if h.qcount == 1 && h.ppost < 2 {
			fmt.Fprintf(w, "abyssinia\n")
			h.quitChan <- true
		}
	case "/metrics":
		h.ptotal++
		h.prom.ServeHTTP(w, r)
		if h.runComplete && !h.waitForQuit {
			h.ppost++
			if h.qcount < 1 && h.ppost == 2 {
				fmt.Printf("zot!\n")
				h.quitChan <- true
			}
		}
	}
}

var promHandler *PromHandler

// PromInit : save parameters for prometheus
func PromInit(port int, wfq bool) {
	switch {
	case port == 0:
		// nop
	case port < 0: // start the handler without tls
		ph := NewPromHandler(-port, wfq)
		walrus.RegisterExitHandler(func() {
			ph.WaitFinish()
		})
		go http.ListenAndServe(fmt.Sprintf(":%d", ph.port), nil)
	default: // config handler to be started after cert's are loaded
		promHandler = NewPromHandler(port, wfq)
	}
}

// PromStartTLS : launch thread for prometheus over tls
func (cert *TLSCert) PromStartTLS() {
	if promHandler == nil {
		return
	}
	// WaitFinish will allow scraping to continue, briefly,
	// after a call to crux.Exit.
	walrus.RegisterExitHandler(func() {
		promHandler.WaitFinish()
	})
	go cert.ListenAndServeTLS(fmt.Sprintf(":%d", promHandler.port), nil)
}

// ListenAndServeTLS : https listener based on flocking certificates.
// we can't use http.Server.TLSConfig because we have our own tls package,
// but the code is essentially cribbed from the go library.
func (cert *TLSCert) ListenAndServeTLS(addr string, handler http.Handler) error {
	if cert == nil {
		return fmt.Errorf("ListenAndServeTLS: nil TLSCert")
	}
	server := &http.Server{Addr: addr, Handler: handler}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	netListener := TCPKeepAliveListener{ln.(*net.TCPListener)}
	tlsConfig := &tls.Config{
		ClientAuth:       tls.RequireAndVerifyClientCert,
		Certificates:     []tls.Certificate{cert.Leaf},
		ClientCAs:        cert.Pool,
		CipherSuites:     []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		CurvePreferences: []tls.CurveID{tls.X25519},
	}
	tlsListener := tls.NewListener(netListener, tlsConfig)
	return server.Serve(tlsListener)
}

// TCPKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections so dead TCP connections (e.g. closing laptop
// mid-download) eventually go away.
// [this is useful, why don't they export it like we do?]

// TCPKeepAliveListener : rhubarb
type TCPKeepAliveListener struct {
	*net.TCPListener
}

// Accept : rhubarb
func (ln TCPKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
