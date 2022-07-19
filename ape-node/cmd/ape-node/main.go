package main

import (
	"flag"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dynm/ape-safer/apebackend"
	"github.com/dynm/ape-safer/apeevm"
	"github.com/dynm/ape-safer/apetxpool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/kataras/golog"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

func defaultKeyCacheFilePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Panic(err)
	}
	cacheDir = path.Join(cacheDir, "ApeSafer")
	err = os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		log.Panic(err)
	}
	fileName := "scratchPadKeyCache.txt"
	return path.Join(cacheDir, fileName)
}

func main() {
	account := flag.String("account", "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "impersonate account")
	upstreamURL := flag.String("upstream", "http://tractor.local:8545", "upstream node")
	listenURL := flag.String("listen", "127.0.0.1:10545", "web3provider bind address port")

	keyCacheFilePath := flag.String("keycache", defaultKeyCacheFilePath(), "state key cache file path")
	batchSize := flag.Int("batchsize", 100, "batch request size")
	logPath := flag.String("logpath", "./ape-node.log", "path to log file")
	debugLevel := flag.String("debug", "info", "debug level")
	flag.Parse()

	pathToLog := *logPath
	// pathToLog += ".%Y%m%d%H%M.log"
	rl, err := rotatelogs.New(
		pathToLog,
		rotatelogs.WithMaxAge(time.Hour*72),
	)
	if err != nil {
		golog.Fatal(err)
	}
	if *logPath == "" {
		myLogger := log.New(os.Stdout, "", 0)
		golog.InstallStd(myLogger)
	}
	golog.SetOutput(rl)
	golog.SetTimeFormat("2006/01/02 15:04:05.000000")
	golog.SetLevel(*debugLevel)

	splittedListen := strings.Split(*listenURL, ":")
	listenAddr := splittedListen[0]
	listenPort, err := strconv.Atoi(splittedListen[1])
	if err != nil {
		golog.Fatal(err)
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
	apeEVM := apeevm.NewApeEVM(*upstreamURL, impersonatedAccount, *keyCacheFilePath, *batchSize)
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
