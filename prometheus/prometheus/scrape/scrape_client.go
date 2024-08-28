// Copyright 2016 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scrape

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	config_util "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"

	"github.com/prometheus/prometheus/config"

	"golang.org/x/net/dns/dnsmessage"
)

// NewScrapeClientFromConfig returns a new HTTP client configured for the
// given ScrapeConfig.
func NewScrapeClientFromConfig(cfg *config.ScrapeConfig) (*http.Client, error) {
	staticConfigs := cfg.ServiceDiscoveryConfig.StaticConfigs
	dnsMap := make(map[string]net.IP)
	for _, tg := range staticConfigs {
		var target, addr string
		if len(tg.Targets) > 0 {
			target = string(tg.Targets[0][model.AddressLabel])
			target = strings.Split(target, ":")[0]
		}
		if target == "" {
			return nil, errors.Errorf("job %s: static config with no targets", cfg.JobName)
		}
		if addr = string(tg.Labels["IPv4"]); addr == "" {
			continue
		}
		ip := net.ParseIP(addr).To4()
		if ip == nil {
			return nil, errors.Errorf("job %s: bad address string \"%s\"", cfg.JobName, addr)
		}
		dnsMap[target] = ip
	}
	if len(dnsMap) == 0 {
		return config_util.NewClientFromConfig(cfg.HTTPClientConfig, cfg.JobName)
	}
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
	rt, err := config_util.NewRoundTripperFromConfigWithDialer(cfg.HTTPClientConfig, cfg.JobName, dialer)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: rt}, nil
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
		return 0, errors.New("cannot parse DNS request")
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
