package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

const (
	NOTIFY_TYPE_NONE = iota
	NOTIFY_TYPE_TX
	NOTIFY_TYPE_ADMIN
)

type NotifyMessage struct {
	MessageType int
	AddressFrom string
	AddressTo   string
	Amount      *big.Int
	Memo        string
	TxHash      string
	BlockTime   int64
}

var (
	fDebug      bool
	fConfigFile string
	packHash    string

	buildVer  = false
	commitID  string
	buildTime string
)

func init() {
	flag.BoolVar(&fDebug, "debug", true, "Debug")
	flag.StringVar(&fConfigFile, "cfg", "config.ini", "Configuration file")
	flag.BoolVar(&buildVer, "version", false, "print build version and then exit")
	flag.StringVar(&packHash, "pack", "", "packet the hash to system")
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(time.Second)
	newBlockTicker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer newBlockTicker.Stop()

	var last_id uint64

	flag.Parse()

	if len(commitID) > 7 {
		fmt.Printf("Build: %s\tLastCommit: %s\n", buildTime, commitID[:7])
	}
	if buildVer {
		os.Exit(0)
	}

	config, err := LoadConfiguration(fConfigFile)
	if err != nil {
		panic(err)
	}

	last_id = config.LastBlock

	r := mux.NewRouter()
	r.HandleFunc("/getMemo", GetMemoHandler(config))
	r.HandleFunc("/getBalance", GetBalanceHandler(config))
	r.HandleFunc("/sendEos", SendEosHandler(config))
	r.HandleFunc("/prepareTrezorEosSign", PrepareTrezorEosSignHandler(config))
	r.HandleFunc("/sendSignedEosTx", SendSignedEosTxHandler(config))

	r.NotFoundHandler = http.HandlerFunc(NotFoundHandler)
	log.Println("last block: ", last_id)

	ch1 := make(chan NotifyMessage, 1024)
	ch2 := make(chan ObjMessage, 1024)
	go Notifier(config, ch1)
	go Listener(config, ch2, ch1, last_id)

	host := ":" + strconv.FormatInt(int64(config.Port), 10)
	log.Printf("Starting web server at %s ...\n", host)

	server := &http.Server{
		ReadTimeout:  time.Duration(30) * time.Second,
		WriteTimeout: time.Duration(30) * time.Second,
		Handler:      r,
	}

	var listener net.Listener
	if listener, err = net.Listen("tcp", host); err != nil {
		log.Println("listen err:", err)
		return
	}
	go server.Serve(listener)

	//launch the signal once avoiding waiting for a long time
	GetNewerBlock(config, ch2)

	stop := 0
	for {
		select {
		case <-ticker.C:
		case <-interrupt:
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			// stop all modules
			//http.StopHttpService(serviceObj)
			stop = 1
			break
		case <-newBlockTicker.C:
			SaveConfiguration(config, fConfigFile)
			if len(ch2) == 0 {
				GetNewerBlock(config, ch2)
			}
		}

		if stop == 1 {
			break
		}
	}
	server.Close()
	SaveConfiguration(config, fConfigFile)
	log.Println("bye")
}
