package dbgpxy

import (
	"net"
	"sync"

	"github.com/oxtoacart/bpool"
	log "github.com/sirupsen/logrus"
)

// TCPServer define a server that listening for TCP connection
type TCPServer interface {
	ListenAndServe() error
	Shutdown()
}

type tcpServerBase struct {
	name        string
	address     string
	listener    *net.TCPListener
	connHandler func(conn net.Conn)
	bytePool    *bpool.BytePool
	quit        chan struct{}
	wg          *sync.WaitGroup
}

func (s *tcpServerBase) configure(name string, address string, connHandler func(conn net.Conn), wg *sync.WaitGroup) {
	s.name = name
	s.address = address
	s.connHandler = connHandler
	s.bytePool = bpool.NewBytePool(1<<16, 1024)
	s.quit = make(chan struct{}, 1)
	s.wg = wg
}

// ListenAndServe listen and handle a new TCP connection (blocking)
func (s *tcpServerBase) ListenAndServe() error {
	logCtx := log.WithFields(log.Fields{"name": s.name, "listen-address": s.address})

	addr, err := net.ResolveTCPAddr("tcp", s.address)
	if err != nil {
		logCtx.WithError(err).Fatal("failed to resolve tcp addr")
		return err
	}

	s.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		logCtx.WithError(err).Fatal("failed to listen port")
		return err
	}

	logCtx.Info("ready for connection")
loop:
	for {
		select {
		case <-s.quit:
			log.WithField("name", s.name).Debug("quit")
			break loop
		default:
			conn, err := s.listener.AcceptTCP()
			if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
				continue
			}
			if err != nil {
				logCtx.WithError(err).Warn("error while accepting connection")
				continue
			}
			go func() {
				s.wg.Add(1)
				s.connHandler(conn)
				s.wg.Done()
			}()
		}
	}

	return nil
}

// Shutdown stop listening for a TCP connection
func (s *tcpServerBase) Shutdown() {
	log.WithField("name", s.name).Debug("shutting down")
	close(s.quit)
	_ = s.listener.Close()
}
