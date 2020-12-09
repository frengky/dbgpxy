package dbgpxy

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/charmap"
)

type initialPacket struct {
	AppID           string `xml:"appid,attr"`
	IDEKey          string `xml:"idekey,attr"`
	Session         string `xml:"session,attr"`
	Thread          string `xml:"thread,attr"`
	Parent          string `xml:"parent,attr"`
	Language        string `xml:"language,attr"`
	ProtocolVersion string `xml:"protocol_version,attr"`
	FileURI         string `xml:"fileuri,attr"`
}

// NewXDebugHandlerServer create new XDebug handler server
func NewXDebugHandlerServer(listenAddress string, ideRepository IDERepository, wg *sync.WaitGroup) *XDebugHandlerServer {
	server := &XDebugHandlerServer{
		ideRepo: ideRepository,
		packetPool: sync.Pool{
			New: func() interface{} {
				return new(initialPacket)
			},
		},
	}

	server.configure("xdebug_handler_server", listenAddress, server.handle, wg)
	return server
}

// XDebugHandlerServer handle a debugging session initialized from XDebug
type XDebugHandlerServer struct {
	ideRepo    IDERepository
	packetPool sync.Pool
	tcpServerBase
}

func (x *XDebugHandlerServer) handle(conn net.Conn) {
	clientIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	logCtx := log.WithFields(log.Fields{"name": x.name, "client": clientIP})
	logCtx.Debug("accepting new connection")

	b := x.bytePool.Get()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer func() {
		x.bytePool.Put(b)
		cancel()
		conn.Close()
	}()

	n, err := conn.Read(b)
	if err != nil {
		logCtx.WithError(err).Error("error while reading data from peer")
		return
	}

	init, err := x.parseInitialPacket(b)
	if err != nil {
		logCtx.WithError(err).Error("error while parsing initial packet")
		return
	}

	logCtx = logCtx.WithField("ide-key", init.IDEKey)
	logCtx.WithField("file-uri", init.FileURI).Info("debugging packet received")

	ide, err := x.ideRepo.FindByKey(init.IDEKey)
	if err != nil {
		logCtx.WithError(err).Warn("ide not registered")
		return
	}

	var d net.Dialer
	var ideConn net.Conn
	ideConn, err = d.DialContext(ctx, "tcp", ide.GetAddress())

	if err != nil {
		logCtx.WithError(err).Warn("unable to contact the ide")
		return
	}

	_, err = ideConn.Write(b[:n])
	if err != nil {
		logCtx.WithError(err).Warn("error while writing data to ide")
		return
	}

	once := sync.Once{}
	cp := func(dst io.WriteCloser, src io.ReadCloser) {
		buf := x.bytePool.Get()
		defer x.bytePool.Put(buf)

		io.CopyBuffer(dst, src, buf)
		once.Do(func() {
			dst.Close()
			src.Close()
		})
	}

	x.wg.Add(1)
	go func() {
		cp(conn, ideConn)
		x.wg.Done()
	}()
	cp(ideConn, conn)
}

func (x *XDebugHandlerServer) parseInitialPacket(subject []byte) (*initialPacket, error) {
	delimiter := []byte("\000")

	first := bytes.Index(subject, delimiter)
	if first < 0 {
		return nil, fmt.Errorf("invalid packet format")
	}
	last := bytes.LastIndex(subject, delimiter)
	if last == first {
		return nil, fmt.Errorf("invalid packet format")
	}

	content := subject[first+1 : last]
	packet := x.packetPool.Get().(*initialPacket)
	defer x.packetPool.Put(packet)

	decoder := xml.NewDecoder(bytes.NewReader(content))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	}

	if err := decoder.Decode(packet); err != nil {
		return nil, err
	}

	return packet, nil
}
