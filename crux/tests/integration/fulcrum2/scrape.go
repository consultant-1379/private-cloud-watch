package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/erixzone/crux/pkg/x509ca"
	"github.com/erixzone/crypto/pkg/x509"

	"golang.org/x/net/dns/dnsmessage"
)

func main() {
	vflag := flag.Bool("v", false, "show metrics (verbose)")
	xflag := flag.Bool("x", false, "don't offer client certificate")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s [-vx] file.yaml [url...]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(127)
	}
	certFile, dnsMap := readConfig(flag.Arg(0))
	if len(dnsMap) == 0 {
		panicOn(fmt.Errorf("empty dnsMap"))
	}
	tlsConfig, err := x509ca.TLSConfigTar(certFile)
	panicOn(err)
	if flag.NArg() < 2 {
		fmt.Printf("certFile: \"%s\"\n", certFile)
		fmt.Printf("dnsMap: %v\n", dnsMap)
		os.Exit(0)
	}

	if *xflag {
		tlsConfig.GetClientCertificate = nil
	}
	tlsConfig.VerifyPeerCertificate = verifyPeerCertificate
	resolver := new(net.Resolver)
	resolver.PreferGo = true
	resolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		return newLocalDNSConn(dnsMap)
	}
	// this dialer is the one from http.DefaultTransport,
	// where it is buried in an anonymous struct.
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
		Resolver:  resolver,
	}
	transport := *http.DefaultTransport.(*http.Transport)
	transport.DialTLS = tlsConfig.DialWithDialerFn(dialer)
	client := &http.Client{Transport: &transport}

	for _, url := range flag.Args()[1:] {
		err := fetchURL(client, url, *vflag)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
	}
	os.Exit(0)
}

func verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	for i, chain := range verifiedChains {
		fmt.Printf("# chain %d: length %d\n", i, len(chain))
		for j, cert := range chain {
			fmt.Printf("# %2d: %s %s\n", j, cert.SerialNumber.Text(16), cert.Subject)
		}
	}
	return nil
}

func fetchURL(client *http.Client, url string, vflag bool) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	fmt.Printf("# resp=%+v\n", resp)
	var total int
	for {
		var n int
		buf := make([]byte, 4096)
		n, err = resp.Body.Read(buf)
		if n > 0 {
			total += n
			if vflag {
				os.Stdout.Write(buf[:n])
			}
		}
		if err != nil {
			break
		}
	}
	fmt.Printf("# resp body: %d bytes\n", total)
	if err != io.EOF {
		return err
	}
	return resp.Body.Close()
}

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

// n.b. this is not a yaml parser, it just uses regexp.

func readConfig(fname string) (certFile string, dnsMap map[string]net.IP) {
	re := regexp.MustCompile(`(ca_file|targets|IPv4):([^"]*"([^"]+)")?`)
	fp, err := os.Open(fname)
	panicOn(err)
	defer fp.Close()
	var target string
	dnsMap = make(map[string]net.IP)
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := scanner.Text()
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		mN := m[len(m)-1]
		switch m[1] {
		case "ca_file":
			s := strings.Split(line, " ")
			n := len(s)
			if n < 2 {
				panicOn(fmt.Errorf("bad ca_file spec"))
			}
			certFile = s[n-1]
		case "targets":
			target = strings.Split(mN, ":")[0]
		case "IPv4":
			ip := net.ParseIP(mN).To4()
			if ip == nil {
				panicOn(fmt.Errorf("bad address string %s", mN))
			}
			dnsMap[target] = ip
		}
	}
	panicOn(scanner.Err())
	return certFile, dnsMap
}

type localDNSConn struct {
	sync.Mutex
	dnsMap map[string]net.IP
	resp   chan []byte
	msg    *bytes.Buffer
	closed bool
}

func newLocalDNSConn(dnsMap map[string]net.IP) (*localDNSConn, error) {
	c := localDNSConn{dnsMap: dnsMap, resp: make(chan []byte, 2)}
	return &c, nil
}

func (c *localDNSConn) Read(b []byte) (int, error) {
	if c.msg == nil || c.msg.Len() == 0 {
		if data, ok := <-c.resp; ok {
			c.msg = bytes.NewBuffer(data)
		} else {
			return 0, io.EOF
		}
	}
	return c.msg.Read(b)
}

func (c *localDNSConn) Write(b []byte) (int, error) {
	c.Lock()
	defer c.Unlock()
	if c.closed {
		return 0, io.EOF
	}
	msg := new(dnsmessage.Message)
	err := msg.Unpack(b[2:])
	if err != nil {
		return 0, fmt.Errorf("cannot parse DNS request")
	}
	if msg.Header.Response || len(msg.Answers) > 0 ||
		len(msg.Authorities) > 0 || len(msg.Additionals) > 0 {
		return len(b), nil
	}
	msg.Header.Response = true
	msg.Header.Authoritative = true
	for _, q := range msg.Questions {
		if q.Type != dnsmessage.TypeA || q.Class != dnsmessage.ClassINET {
			continue
		}
		name := strings.TrimSuffix(q.Name.String(), ".")
		addr, ok := c.dnsMap[name]
		if !ok {
			continue
		}
		rh := dnsmessage.ResourceHeader{
			Name:  q.Name,
			Class: q.Class,
			TTL:   3600,
		}
		var ar dnsmessage.AResource
		copy(ar.A[:], addr)
		msg.Answers = append(msg.Answers, dnsmessage.Resource{rh, &ar})
	}
	buf := make([]byte, 2, 514)
	buf, err = msg.AppendPack(buf)
	if err != nil {
		return 0, err
	}
	n := len(buf)-2
	buf[0] = byte(n>>8)
	buf[1] = byte(n)
	c.resp <- buf

	return len(b), nil
}

func (c *localDNSConn) Close() (err error) {
	c.Lock()
	defer c.Unlock()
	c.closed = true
	close(c.resp)
	return nil
}

func (c *localDNSConn) LocalAddr() net.Addr {
	return c
}

func (c *localDNSConn) RemoteAddr() net.Addr {
	return c
}

func (c *localDNSConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *localDNSConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *localDNSConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *localDNSConn) Network() string {
	return "gochan"
}

func (c *localDNSConn) String() string {
	return "gochan:dnsMap"
}
