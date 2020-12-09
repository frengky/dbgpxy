package dbgpxy

import (
	"fmt"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// NewIDEHandlerServer create new IDE handler server
func NewIDEHandlerServer(listenAddress string, ideStorage IDEStorage, wg *sync.WaitGroup) *IDEHandlerServer {
	server := &IDEHandlerServer{
		ideStorage: ideStorage,
	}

	server.configure("ide_handler_server", listenAddress, server.handle, wg)
	return server
}

// IDEHandlerServer handle registration request from an IDE
type IDEHandlerServer struct {
	ideStorage IDEStorage
	tcpServerBase
}

func (s *IDEHandlerServer) handle(conn net.Conn) {
	clientIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	logCtx := log.WithFields(log.Fields{"name": s.name, "client": clientIP})
	logCtx.Debug("accepting new connection")

	b := s.bytePool.Get()

	defer func() {
		s.bytePool.Put(b)
		conn.Close()
	}()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Read(b); err != nil {
		logCtx.WithError(err).Error("error while reading data from peer")
		return
	}

	cmdArgs, err := getCommandArgs(string(b))
	if err != nil {
		logCtx.WithError(err).Error("error while parsing command")
		return
	}

	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	args := cmdArgs.GetArguments()
	var werr error

	switch cmd := cmdArgs.GetCommand(); cmd {
	case "proxyinit":
		ideKey, _ := args["k"]
		port, _ := args["p"]

		if s.ideStorage.Has(ideKey) {
			logCtx.WithField("ide-key", ideKey).Warn("ide already exists")
			errReply := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<proxyinit success="0"><error id="0"><message>ide key already exists</message></error></proxyinit>`)
			_, werr = conn.Write(errReply)
			break
		}

		logCtx.WithField("ide-key", ideKey).Info("registering ide")
		s.ideStorage.Put(ideKey, NewIDE(ideKey, clientIP, port))

		successReply := []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<proxyinit success="1" idekey="%s" address="%s" port="%s"/>`, ideKey, clientIP, port))
		_, werr = conn.Write(successReply)

	case "proxystop":
		ideKey, _ := args["k"]

		if !s.ideStorage.Has(ideKey) {
			logCtx.WithField("ide-key", ideKey).Warn("ide is not exists")
			errReply := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<proxystop success="0"><error id="0"><message>ide key already exists</message></error></proxystop>`)
			_, werr = conn.Write(errReply)
			break
		}

		logCtx.WithField("ide-key", ideKey).Info("unregistering ide")
		s.ideStorage.Forget(ideKey)

		successReply := []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<proxystop success="1" idekey="%s"/>`, ideKey))
		_, werr = conn.Write(successReply)

	default:
		logCtx.WithField("command", cmd).Warn("unsupported command")
		errReply := []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<%s success="0"><error id="0"><message>Unsupported command</message></error></%s>`, cmd, cmd))
		_, werr = conn.Write(errReply)
		return
	}

	if werr != nil {
		logCtx.WithError(werr).Error("error while writing data to peer")
	}
}
