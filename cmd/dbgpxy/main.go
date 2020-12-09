package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/frengky/dbgpxy"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	register string
	debug    string
	ide      string

	wg = &sync.WaitGroup{}

	rootCmd = &cobra.Command{
		Use:   "dbgpxy",
		Short: "Xdebug DBGp proxy",
		Long:  "Xdebug DBGp proxy written in Go",
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
)

func init() {
	log.SetLevel(log.DebugLevel)

	rootCmd.PersistentFlags().StringVar(&ide, "ide", "", "Forwards all to an IDE, example: 127.0.0.1:9000")
	rootCmd.PersistentFlags().StringVarP(&register, "register", "r", "", "Listen for IDE registration, example: 0.0.0.0:9033")
	rootCmd.PersistentFlags().StringVarP(&debug, "debug", "d", "", "Listen for XDebug, example: 0.0.0.0:9003")
	rootCmd.MarkPersistentFlagRequired("debug")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run() {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

	var ideHandlerServer *dbgpxy.IDEHandlerServer
	var ideRepository dbgpxy.IDERepository

	if ide != "" {
		ideHost, idePort, err := net.SplitHostPort(ide)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid ide address %s", ide)
			os.Exit(1)
		}
		ideRepository = newStaticIDERepository("STATIC", ideHost, idePort)
	} else if register != "" {
		ideStorage := dbgpxy.NewIDEStorage()
		ideRepository = dbgpxy.NewIDERepository(ideStorage)
		ideHandlerServer = dbgpxy.NewIDEHandlerServer(register, ideStorage, wg)

		wg.Add(1)
		go func() {
			ideHandlerServer.ListenAndServe()
			wg.Done()
		}()
	} else {
		fmt.Fprint(os.Stderr, "Error: please enable ide registration or specify ide address")
		os.Exit(1)
	}

	log.Info("starting debugging proxy")
	xdebugHandlerServer := dbgpxy.NewXDebugHandlerServer(debug, ideRepository, wg)

	wg.Add(1)
	go func() {
		xdebugHandlerServer.ListenAndServe()
		wg.Done()
	}()

	<-sigChan
	if ideHandlerServer != nil {
		ideHandlerServer.Shutdown()
	}
	xdebugHandlerServer.Shutdown()
	wg.Wait()
	os.Exit(0)
}

type staticIDERepository struct {
	ide dbgpxy.IDE
}

func newStaticIDERepository(key string, ip string, port string) *staticIDERepository {
	return &staticIDERepository{
		ide: dbgpxy.NewIDE(key, ip, port),
	}
}

func (s *staticIDERepository) FindByKey(key string) (dbgpxy.IDE, error) {
	log.WithFields(log.Fields{
		"ide-key":     key,
		"dst-address": s.ide.GetAddress(),
	}).Debug("redirect debugging packet")
	return s.ide, nil
}
