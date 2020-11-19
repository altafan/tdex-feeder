// Copyright (c) 2020 The VulpemVentures developers

// Feeder allows to connect an external price feed to the TDex Daemon to determine the current market price.
package main

import (
	"flag"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
	"github.com/tdex-network/tdex-feeder/config"
	"github.com/tdex-network/tdex-feeder/pkg/conn"
	"github.com/tdex-network/tdex-feeder/pkg/marketinfo"

	pboperator "github.com/tdex-network/tdex-protobuf/generated/go/operator"
)

const (
	defaultConfigPath = "./config/config.json"
)

func main() {

	// Checks for command line flags for Config Path.
	confFlag := flag.String("conf", defaultConfigPath, "Configuration File Path")
	debugFlag := flag.String("debug", "false", "Log Debug Informations")
	flag.Parse()

	if *debugFlag == "true" {
		log.SetLevel(log.DebugLevel)
	}
	// Loads Config File.
	conf, err := config.LoadConfig(*confFlag)
	if err != nil {
		log.Fatal(err)
	}

	// Interrupt Notification.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Dials the connection the the Socket.
	cSocket, err := conn.ConnectToSocket(conf.KrakenWsEndpoint)
	if err != nil {
		log.Fatal("Socket Connection Error: ", err)
	}
	defer cSocket.Close()

	// Set up the connection to the gRPC server.
	conngRPC, err := conn.ConnectTogRPC(conf.DaemonEndpoint)
	if err != nil {
		log.Fatal("gRPC Connection Error: ", err)
	}
	defer conngRPC.Close()
	clientgRPC := pboperator.NewOperatorClient(conngRPC)

	// Loads Config Markets infos into Data Structure and Subscribes to
	// Messages from this Markets.
	numberOfMarkets := len(conf.Markets)
	marketsInfos := make([]*marketinfo.MarketInfo, numberOfMarkets)
	for i, marketConfig := range conf.Markets {
		marketsInfos[i] = marketinfo.InitialMarketInfo(marketConfig)
		defer marketsInfos[i].GetTicker().Stop()
		m := conn.CreateSubscribeToMarketMessage(marketConfig.KrakenTicker)
		err = conn.SendRequestMessage(cSocket, m)
		if err != nil {
			log.Fatal("Couldn't send request message: ", err)
		}
	}

	// Gets messages from subscriptions.
	done := make(chan string)
	go conn.GetMessages(done, cSocket, marketsInfos)

	// Loop to keep cycle alive. Periodically sends gRPC request to
	// update price. Waits for Interrupt to close the connection.
	for {
		for _, marketInfo := range marketsInfos {
			select {
			case <-marketInfo.GetTicker().C:
				// Sends gRPC request to update price
				err := conn.UpdateMarketPricegRPC(marketInfo, clientgRPC)
				if err != nil {
					log.Println("Couldn't send gRPC request: ", err)
				}
			case <-interrupt:
				log.Println("Shutting down Feeder")
				err := cSocket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					log.Fatal("write close:", err)
				}
				select {
				case <-done:
				case <-time.After(time.Second):
				}
				return
			}
		}
	}
}
