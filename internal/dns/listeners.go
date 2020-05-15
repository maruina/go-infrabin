package dns

import (
	"net"
	"sync"
	"time"

	miekg "github.com/miekg/dns"
)

func runLocalTCPServerWithFinChan(laddr string) (*miekg.Server, string, chan error, error) {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, "", nil, err
	}

	server := &miekg.Server{Listener: l, ReadTimeout: time.Hour, WriteTimeout: time.Hour}

	waitLock := sync.Mutex{}
	waitLock.Lock()
	server.NotifyStartedFunc = waitLock.Unlock

	// fin must be buffered so the goroutine below won't block
	// forever if fin is never read from. If something isn't interested
	// in our errors we should still close our underlying network socket
	fin := make(chan error, 2)
	go func() {
		fin <- server.ActivateAndServe()
		fin <- l.Close()
	}()

	waitLock.Lock()
	return server, l.Addr().String(), fin, nil
}

func runLocalUDPServerWithFinChan(laddr string) (*miekg.Server, string, chan error, error) {
	pc, err := net.ListenPacket("udp", laddr)
	if err != nil {
		return nil, "", nil, err
	}
	server := &miekg.Server{PacketConn: pc, ReadTimeout: time.Hour, WriteTimeout: time.Hour}

	waitLock := sync.Mutex{}
	waitLock.Lock()
	server.NotifyStartedFunc = waitLock.Unlock

	// fin must be buffered so the goroutine below won't block
	// forever if fin is never read from. If something isn't interested
	// in our errors we should still close our underlying network socket
	fin := make(chan error, 2)
	go func() {
		fin <- server.ActivateAndServe()
		fin <- pc.Close()
	}()

	waitLock.Lock()
	return server, pc.LocalAddr().String(), fin, nil
}
