package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get the absolute path to prevent directory traversal
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		// if we failed to get the absolute path respond with a 400 bad request
		// and stop
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// prepend the path with the path to the static directory
	path = filepath.Join(h.staticPath, path)

	// check whether a file exists at the given path
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

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

	stack.RegisterAPIs(apebackend.GetApeAPIs(b))
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
