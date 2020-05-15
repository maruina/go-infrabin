package dns

import (
	"fmt"
	"log"

	miekg "github.com/miekg/dns"
)

func init() {
	// Register patterns for dns listners
	miekg.HandleFunc("infrabin.com.", ARecordResponseLoopback)
	miekg.HandleFunc("arecord.infrabin.com", ARecordResponseLoopback)
	miekg.HandleFunc("aaaarecord.infrabin.com", AAAARecordResponseLoopback)
	miekg.HandleFunc("cname.infrabin.com", CNAMERecordResponse)
}

var (
	tcpServer *miekg.Server
	udpServer *miekg.Server
)

// Runs a TCP and UDP DNS Server on port {port}
func RunDefaultDNSServerWithFinChan(port int) (chan struct{}, chan struct{}) {
	// our term channel
	term := make(chan struct{})

	// our done channel
	done := make(chan struct{})

	go func() {
		s, tcpServerAddress, tcpFinish, err := runLocalTCPServerWithFinChan(fmt.Sprintf(":%d", port))
		if err != nil {
			panic(err)
		}
		// Register TCP Server so we can restart it
		tcpServer = s
		log.Printf("TCP DNS Server started on Address: %v\n", tcpServerAddress)

		s, udpServerAddress, udpFinish, err := runLocalUDPServerWithFinChan(fmt.Sprintf(":%d", port))
		if err != nil {
			panic(err)
		}
		// Register UDP Server so we can restart it
		udpServer = s
		log.Printf("UDP DNS Server started on Address: %v\n", udpServerAddress)

		// Select over all our signals.
		select {
		case <-term:
			log.Print("Shutting down DNS listners")
			_ = tcpServer.Shutdown()
			_ = udpServer.Shutdown()
			done <- struct{}{}
			return
		case err := <-tcpFinish:
			log.Printf("TCP DNS Server Died unexpectedly! restarting: %v\n", err.Error())
			_ = tcpServer.Shutdown()
			tcpServer, tcpServerAddress, tcpFinish, err = runLocalTCPServerWithFinChan(fmt.Sprintf(":%d", port))

		case err := <-udpFinish:
			log.Printf("UDP DNS Server Died unexpectedly! restarting: %v\n", err.Error())
			_ = udpServer.Shutdown()
			tcpServer, udpServerAddress, udpFinish, err = runLocalUDPServerWithFinChan(fmt.Sprintf(":%d", port))
		}
	}()

	return term, done
}
