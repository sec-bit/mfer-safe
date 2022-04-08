package main

import (
	"flag"
	"log"
	"strconv"
	"strings"

	"github.com/dynm/ape-safer/apebackend"
	"github.com/dynm/ape-safer/apeevm"
	"github.com/dynm/ape-safer/apetxpool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

func main() {
	account := flag.String("account", "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "impersonate account")
	upstreamURL := flag.String("upstream", "http://tractor.local:8545", "upstream node")
	listenURL := flag.String("listen", "127.0.0.1:10545", "web3provider bind address port")
	flag.Parse()

	splittedListen := strings.Split(*listenURL, ":")
	listenAddr := splittedListen[0]
	listenPort, err := strconv.Atoi(splittedListen[1])
	if err != nil {
		log.Panic(err)
	}

	stack, err := node.New(&node.Config{
		Name: "ape-safer",
		P2P: p2p.Config{
			NoDial:     true,
			ListenAddr: "",
		},
		HTTPHost:         listenAddr,
		HTTPPort:         listenPort,
		HTTPCors:         []string{"*"},
		HTTPVirtualHosts: []string{"*"},
	})
	if err != nil {
		log.Panic(err)
	}

	impersonatedAccount := common.HexToAddress(*account)
	apeEVM := apeevm.NewApeEVM(*upstreamURL, impersonatedAccount)
	txPool := apetxpool.NewApeTxPool()
	b := apebackend.NewApeBackend(apeEVM, txPool, impersonatedAccount)

	stack.RegisterAPIs(apebackend.GetEthAPIs(b))
	if err := stack.Start(); err != nil {
		log.Panic(err)
	}

	selfRPCClient, err := stack.Attach()
	if err != nil {
		log.Panic(err)
	}

	apeEVM.SelfClient = selfRPCClient
	apeEVM.SelfConn = ethclient.NewClient(selfRPCClient)

	select {}
}
