package apestate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/dynm/ape-safer/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/kataras/golog"
	"github.com/tj/go-spin"
)

type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}
type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

type StorageReq struct {
	Address common.Address
	Key     common.Hash
	Value   common.Hash
	Error   error
}

func (r *StorageReq) Hash() common.Hash {
	return crypto.Keccak256Hash(r.Address.Bytes(), r.Key.Bytes())
}

type OverlayState struct {
	ctx             context.Context
	ec              *rpc.Client
	conn            *ethclient.Client
	parent          *OverlayState
	bn              int64
	lastBN          int64
	scratchPadMutex *sync.RWMutex
	scratchPad      map[string][]byte
	cacheFilePath   string
	batchSize       int

	accessedAccountsMutex *sync.RWMutex
	accessedAccounts      map[common.Address]bool

	logs                            []*types.Log
	txLogs                          map[common.Hash][]*types.Log
	receipts                        map[common.Hash]*types.Receipt
	currentTxHash, currentBlockHash common.Hash
	deriveCnt                       int64
	rpcCnt                          int64
	storageReqChan                  chan chan StorageReq
	accReqChan                      chan chan FetchedAccountResult

	loadAccountMutex *sync.Mutex

	upstreamReqCh chan bool
	clientReqCh   chan bool

	shouldStop          chan bool
	shoudRevertSnapshot chan bool
	reason              string

	stateID uint64
}

func NewOverlayState(ctx context.Context, ec *rpc.Client, bn int64, keyCacheFilePath string, batchSize int) *OverlayState {
	state := &OverlayState{
		ctx:             ctx,
		ec:              ec,
		conn:            ethclient.NewClient(ec),
		parent:          nil,
		bn:              bn,
		scratchPadMutex: &sync.RWMutex{},
		scratchPad:      make(map[string][]byte),
		cacheFilePath:   keyCacheFilePath,
		batchSize:       batchSize,

		accessedAccountsMutex: &sync.RWMutex{},
		accessedAccounts:      make(map[common.Address]bool),

		txLogs:           make(map[common.Hash][]*types.Log),
		logs:             make([]*types.Log, 0),
		receipts:         make(map[common.Hash]*types.Receipt),
		deriveCnt:        0,
		storageReqChan:   make(chan chan StorageReq, 500),
		accReqChan:       make(chan chan FetchedAccountResult, 200),
		loadAccountMutex: &sync.Mutex{},

		upstreamReqCh: make(chan bool, 100),
		clientReqCh:   make(chan bool, 100),

		shouldStop: make(chan bool),
	}
	go state.timeSlot()
	// go state.statistics()
	return state
}

func (s *OverlayState) Derive(reason string) *OverlayState {
	state := &OverlayState{
		parent:           s,
		scratchPad:       make(map[string][]byte),
		txLogs:           make(map[common.Hash][]*types.Log),
		logs:             make([]*types.Log, len(s.logs)),
		receipts:         make(map[common.Hash]*types.Receipt),
		deriveCnt:        s.deriveCnt + 1,
		currentTxHash:    s.currentTxHash,
		currentBlockHash: s.currentBlockHash,

		stateID:    rand.Uint64(),
		shouldStop: s.shouldStop,
		reason:     reason,
	}
	golog.Debugf("derive from: %s, id: %02x, depth: %d", reason, state.stateID, state.deriveCnt)
	copy(state.logs, s.logs)
	for k, v := range s.receipts {
		state.receipts[k] = v
	}
	// go state.printStateID()
	return state
}

func (s *OverlayState) Parent() *OverlayState {
	// s.scratchPad = make(map[string][]byte)
	golog.Debugf("poping id: %d, reason: %s", s.stateID, s.reason)
	// close(s.shouldStop)
	return s.parent
}

func (s *OverlayState) printStateID() {
	for {
		select {
		// case <-time.After(time.Second * 2):
		// 	parentID := uint64(0)
		// 	if s.parent == nil {
		// 		parentID = 0
		// 	} else {
		// 		parentID = s.parent.stateID
		// 	}
		// 	log.Printf("stateID: %02x, parentID: %02x, reason: %s", s.stateID, parentID, s.reason)
		case <-s.shouldStop:
			s.scratchPad = nil
			s.logs = nil
			s.txLogs = nil
			log.Printf("stateID: %02x, reason: %s STOPPED", s.stateID, s.reason)
			return
		case <-s.shoudRevertSnapshot:
			s.scratchPad = nil
			s.logs = nil
			s.txLogs = nil
			log.Printf("stateID: %02x, reason: %s STOPPED [revert snapshot]", s.stateID, s.reason)
			return
		}
	}
}

type RequestType int

const (
	GET_BALANCE RequestType = iota
	GET_NONCE
	GET_CODE
	GET_CODEHASH
	GET_STATE
)

var (
	BALANCE_KEY  = crypto.Keccak256Hash([]byte("apesafer-scratchpad-balance"))
	NONCE_KEY    = crypto.Keccak256Hash([]byte("apesafer-scratchpad-nonce"))
	CODE_KEY     = crypto.Keccak256Hash([]byte("apesafer-scratchpad-code"))
	CODEHASH_KEY = crypto.Keccak256Hash([]byte("apesafer-scratchpad-codehash"))
	STATE_KEY    = crypto.Keccak256Hash([]byte("apesafer-scratchpad-state"))
	SUICIDE_KEY  = crypto.Keccak256Hash([]byte("apesafer-suicide-state"))
)

type FetchedAccountResult struct {
	Account  common.Address
	Balance  hexutil.Big
	CodeHash common.Hash
	Nonce    hexutil.Uint64
	Code     hexutil.Bytes
}

func (s *OverlayState) loadAccountBatchRPC(accounts []common.Address) ([]FetchedAccountResult, error) {
	rpcTries := 0
	bn := big.NewInt(s.bn)
	hexBN := hexutil.EncodeBig(bn)

	result := make([]FetchedAccountResult, len(accounts))
	batchElem := make([]rpc.BatchElem, len(accounts)*3)

	for i, account := range accounts {
		getNonceReq := rpc.BatchElem{
			Method: "eth_getTransactionCount",
			Args:   []interface{}{account, hexBN},
			Result: &result[i].Nonce,
		}

		getBalanceReq := rpc.BatchElem{
			Method: "eth_getBalance",
			Args:   []interface{}{account, hexBN},
			Result: &result[i].Balance,
		}

		getCodeReq := rpc.BatchElem{
			Method: "eth_getCode",
			Args:   []interface{}{account, hexBN},
			Result: &result[i].Code,
		}
		batchElem[i*3] = getNonceReq
		batchElem[i*3+1] = getBalanceReq
		batchElem[i*3+2] = getCodeReq

		s.accessedAccountsMutex.Lock()
		s.accessedAccounts[account] = true
		s.accessedAccountsMutex.Unlock()
	}

	step := s.batchSize
	start := time.Now()
	for begin := 0; begin < len(batchElem); begin += step {
		for {
			// s.upstreamReqCh <- true
			end := begin + step
			if end > len(batchElem) {
				end = len(batchElem)
			}
			golog.Infof("loadAccount batch req(total=%d): begin: %d, end: %d", len(batchElem), begin, end)
			err := s.ec.BatchCallContext(s.ctx, batchElem[begin:end])
			if err != nil {
				rpcTries++
				if rpcTries > 5 {
					return nil, err
				} else {
					golog.Warn("retrying loadAccountSimple")
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}
			break
		}
	}

	for i := range accounts {
		if len(result[i].Code) == 0 {
			result[i].CodeHash = common.Hash{}
		} else {
			result[i].CodeHash = crypto.Keccak256Hash(result[i].Code)
		}
	}
	golog.Infof("fetched %d accounts batched@%d (consumes: %v)", len(accounts), s.bn, time.Since(start))

	return result, nil
}

func (s *OverlayState) loadAccountViaGetProof(account common.Address) (*AccountResult, []byte, error) {
	var result AccountResult
	var code hexutil.Bytes
	rpcTries := 0
	hexBN := hexutil.EncodeBig(big.NewInt(int64(s.bn)))

	getProofReq := rpc.BatchElem{
		Method: "eth_getProof",
		Args:   []interface{}{account, []string{}, hexBN},
		Result: &result,
	}

	getCodeReq := rpc.BatchElem{
		Method: "eth_getCode",
		Args:   []interface{}{account, hexBN},
		Result: &code,
	}

	for {
		start := time.Now()
		err := s.ec.BatchCallContext(s.ctx, []rpc.BatchElem{getProofReq, getCodeReq})
		if err != nil {
			rpcTries++
			if rpcTries > 5 {
				return nil, nil, err
			} else {
				golog.Warn("retrying getProof")
				time.Sleep(100 * time.Millisecond)
				continue
			}
		} else {
			rpcTries = 0
			if getProofReq.Error != nil {
				golog.Errorf("getProof err: %v", getProofReq)
			}
			if getCodeReq.Error != nil {
				golog.Errorf("getProof err: %v", getCodeReq)
			}
			golog.Infof("fetched account batched@%d {proof, code}: %s (consumes: %v)", s.bn, account.Hex(), time.Since(start))
			break
		}
	}

	return &result, code, nil
}

func (s *OverlayState) loadStateBatchRPC(storageReqs []*StorageReq) error {
	// TODO: dedup

	s.rpcCnt++
	// s.upstreamReqCh <- true
	reqs := make([]rpc.BatchElem, len(storageReqs))
	values := make([]common.Hash, len(storageReqs))
	bn := big.NewInt(s.bn)
	hexBN := hexutil.EncodeBig(bn)
	for i := range reqs {
		reqs[i] = rpc.BatchElem{
			Method: "eth_getStorageAt",
			Args:   []interface{}{storageReqs[i].Address, storageReqs[i].Key, hexBN},
			Result: &values[i],
		}
	}

	// for i:=0;i<len(storageReqs);i+=2500 {

	// }
	step := s.batchSize
	start := time.Now()
	for begin := 0; begin < len(reqs); begin += step {
		end := begin + step
		if end > len(reqs) {
			end = len(reqs)
		}
		golog.Infof("loadState batch req(total=%d): begin: %d, end: %d", len(reqs), begin, end)
		if err := s.ec.BatchCallContext(s.ctx, reqs[begin:end]); err != nil {
			return err
		}
	}

	golog.Infof("fetched %d state batched@%d (consumes: %v)", len(reqs), s.bn, time.Since(start))

	for i := range storageReqs {
		storageReqs[i].Value = values[i]
	}
	return nil
}

func (s *OverlayState) loadStateRPC(account common.Address, key common.Hash) (common.Hash, error) {
	s.rpcCnt++
	// s.upstreamReqCh <- true
	storage, err := s.conn.StorageAt(s.ctx, account, key, big.NewInt(s.bn))
	if err != nil {
		return common.Hash{}, err
	}
	value := common.BytesToHash(storage)
	return value, nil
}

func (s *OverlayState) timeSlot() {
	tickerStorage := time.NewTicker(time.Millisecond * 3)
	tickerAccount := time.NewTicker(time.Millisecond * 10)
	for {
		storageReqLen := len(s.storageReqChan)
		accReqLen := len(s.accReqChan)
		select {
		case <-tickerStorage.C:
			storageReqPending := make([]*StorageReq, storageReqLen)
			storageReqChanPending := make([]chan StorageReq, storageReqLen)
			for i := 0; i < storageReqLen; i++ {
				req := <-s.storageReqChan
				storageReq := <-req
				storageReqPending[i] = &storageReq
				storageReqChanPending[i] = req
			}
			if storageReqLen > 0 {
				for {
					err := s.loadStateBatchRPC(storageReqPending)
					if err != nil {
						golog.Errorf("loadStateBatch, err: %v", err)
						time.Sleep(time.Second * 1)
					} else {
						break
					}
				}
			}

			for i := 0; i < storageReqLen; i++ {
				req := storageReqChanPending[i]
				req <- *storageReqPending[i]
				close(req)
			}
		case <-tickerAccount.C:
			accReqPending := make([]*FetchedAccountResult, accReqLen)
			accReqChanPending := make([]chan FetchedAccountResult, accReqLen)
			accounts := make([]common.Address, accReqLen)
			for i := 0; i < accReqLen; i++ {
				req := <-s.accReqChan
				accReq := <-req
				accReqPending[i] = &accReq
				accReqChanPending[i] = req
				accounts[i] = accReq.Account
			}

			var accResult []FetchedAccountResult
			var err error
			if accReqLen > 0 {
				for {
					accResult, err = s.loadAccountBatchRPC(accounts)
					if err != nil {
						golog.Errorf("loadAccountBatchRPC, err: %v", err)
						time.Sleep(time.Second * 1)
					} else {
						break
					}
				}
			}

			for i := 0; i < len(accResult); i++ {
				req := accReqChanPending[i]
				req <- accResult[i]
				close(req)
			}
		}
	}
}

func (db *OverlayState) statistics() {
	spinnerUpstream := &spin.Spinner{}
	spinnerUpstream.Set("🌍🌎🌏")
	spinnerClient := &spin.Spinner{}
	spinnerClient.Set("▁▂▃▄▅▆▇█▇▆▅▄▃▁")

	upstreamReqCnt := 0
	clientReqCnt := 0
	upstreamSpinStr := "-"
	clientSpinStr := "-"
	statisticsStr := "\nUpstream %s  [%d]\tDownstream %s  [%d]"

	ticker := time.NewTicker(time.Second)
	for {
		fmt.Print("\033[G\033[K")
		select {
		case <-db.upstreamReqCh:
			upstreamReqCnt++
			upstreamSpinStr = spinnerUpstream.Next()
			fmt.Printf(statisticsStr, upstreamSpinStr, upstreamReqCnt, clientSpinStr, clientReqCnt)
		case <-db.clientReqCh:
			clientReqCnt++
			clientSpinStr = spinnerClient.Next()
			fmt.Printf(statisticsStr, upstreamSpinStr, upstreamReqCnt, clientSpinStr, clientReqCnt)
		case <-ticker.C:
			upstreamSpinStr = spinnerUpstream.Next()
			clientSpinStr = spinnerClient.Next()
			fmt.Printf(statisticsStr, upstreamSpinStr, upstreamReqCnt, clientSpinStr, clientReqCnt)
		}
		fmt.Printf("\033[A")
	}
}

func (s *OverlayState) loadState(account common.Address, key common.Hash) common.Hash {
	retChan := make(chan StorageReq)
	s.storageReqChan <- retChan
	retChan <- StorageReq{Address: account, Key: key}
	result := <-retChan
	// spew.Dump(result)
	return result.Value
}

func (s *OverlayState) loadAccount(account common.Address) FetchedAccountResult {
	retChan := make(chan FetchedAccountResult)
	s.accReqChan <- retChan
	retChan <- FetchedAccountResult{Account: account}
	result := <-retChan
	// spew.Dump(result)
	return result
}

func calcKey(op common.Hash, account common.Address) string {
	return string(append(op.Bytes(), account.Bytes()...))
}

func calcStateKey(account common.Address, key common.Hash) string {
	getStateKey := calcKey(STATE_KEY, account)
	stateKey := getStateKey + string(key.Bytes())
	return stateKey
}

func (s *OverlayState) resetScratchPad(bn int64) {
	s.scratchPadMutex.Lock()
	s.bn = bn
	golog.Debug("[reset scratchpad] lock scratchPad")

	f, err := os.OpenFile(s.cacheFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Panicf("openfile error: %v", err)
	}
	defer f.Close()
	fmt.Printf("cache saved @ %s\n", s.cacheFilePath)

	golog.Debug("[reset scratchpad] load cached scratchPad key")
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		s.scratchPad[string(STATE_KEY.Bytes())+string(common.Hex2Bytes(txt))] = []byte{}
		if scanner.Err() != nil {
			break
		}
	}

	cachedStr := ""
	reqs := make([]*StorageReq, 0)
	for key := range s.scratchPad {
		keyBytes := []byte(key)
		if len(keyBytes) == 32+20+32 && common.BytesToHash(keyBytes[:32]) == STATE_KEY {
			acc := common.BytesToAddress(keyBytes[32 : 32+20])
			s.accessedAccountsMutex.Lock()
			s.accessedAccounts[acc] = true
			s.accessedAccountsMutex.Unlock()
			key := common.BytesToHash(keyBytes[32+20:])
			reqs = append(reqs, &StorageReq{Address: acc, Key: key})
			cachedStr += (common.Bytes2Hex(keyBytes[32:]) + "\n")
			if err != nil {
				golog.Errorf("write string err: %v", err)
			}
		}
	}

	err = s.loadStateBatchRPC(reqs)
	if err != nil {
		log.Panic(err)
	}
	f.Truncate(0)
	f.Seek(0, 0)
	f.WriteString(cachedStr)

	for _, result := range reqs {
		stateKey := calcStateKey(result.Address, result.Key)
		s.scratchPad[stateKey] = result.Value[:]
	}

	golog.Infof("[reset scratchpad] state prefetch done, slot num: %d", len(s.scratchPad))
	golog.Infof("[reset scratchpad] prefetching %d accounts", len(s.accessedAccounts))
	accounts := make([]common.Address, 0)
	s.accessedAccountsMutex.RLock()
	for k := range s.accessedAccounts {
		accounts = append(accounts, k)
	}
	s.accessedAccountsMutex.RUnlock()

	accountResults, err := s.loadAccountBatchRPC(accounts)
	if err != nil {
		golog.Errorf("loadAccountBatchRPC failed: %v", err)
		s.scratchPadMutex.Unlock()
		return
	}
	for i := range accountResults {
		nonce := uint64(accountResults[i].Nonce)
		balance := accountResults[i].Balance.ToInt()
		codeHash := accountResults[i].CodeHash
		s.scratchPad[calcKey(BALANCE_KEY, accounts[i])] = balance.Bytes()
		s.scratchPad[calcKey(NONCE_KEY, accounts[i])] = big.NewInt(int64(nonce)).Bytes()
		s.scratchPad[calcKey(CODE_KEY, accounts[i])] = accountResults[i].Code
		s.scratchPad[calcKey(CODEHASH_KEY, accounts[i])] = codeHash.Bytes()
	}
	golog.Info("[reset scratchpad] account prefetch done")

	s.scratchPadMutex.Unlock()
	golog.Debug("[reset scratchpad] unlock scratchPad")

}

func (s *OverlayState) get(account common.Address, action RequestType, key common.Hash) ([]byte, error) {
	if s.parent == nil && s.bn != s.lastBN {
		golog.Infof("State BN: %d", s.bn)
		s.lastBN = s.bn
	}
	var scratchpadKey string
	switch action {
	case GET_BALANCE:
		scratchpadKey = calcKey(BALANCE_KEY, account)
	case GET_NONCE:
		scratchpadKey = calcKey(NONCE_KEY, account)
	case GET_CODE:
		scratchpadKey = calcKey(CODE_KEY, account)
	case GET_CODEHASH:
		scratchpadKey = calcKey(CODEHASH_KEY, account)
	case GET_STATE:
		scratchpadKey = calcStateKey(account, key)
	}

	if s.parent == nil {
		// s.clientReqCh <- true
		s.scratchPadMutex.Lock()
		if val, ok := s.scratchPad[scratchpadKey]; ok {
			// if action == GET_STATE {
			// log.Printf("got state at root, acc[%s].%s=0x%02x\nstateKey: %02x", account.Hex(), key.Hex(), val, scratchpadKey)
			// }
			s.scratchPadMutex.Unlock()
			return val, nil
		}
		s.scratchPadMutex.Unlock()

		var res []byte
		// UPDATE_BN_AND_RETRY:
		switch action {
		case GET_STATE:
			result := s.loadState(account, key)
			// result, err := s.loadState(account, key)
			// if err != nil {
			// 	log.Print(err)
			// 	bn, err := s.conn.BlockNumber(s.ctx)
			// 	if err != nil {
			// 		log.Panic(err)
			// 	}
			// 	log.Printf("Resetting State... BN: %d", bn)
			// 	s.resetScratchPad(int64(bn))
			// 	goto UPDATE_BN_AND_RETRY
			// }

			s.scratchPadMutex.Lock()
			s.scratchPad[scratchpadKey] = result.Bytes()
			// log.Printf("underlying get state, acc[%s].%s=%s", account.Hex(), key.Hex(), result.Hex())
			s.scratchPadMutex.Unlock()
			res = result.Bytes()

		case GET_BALANCE, GET_NONCE, GET_CODE, GET_CODEHASH:
			// log.Printf("underlying get account: %s", account.Hex())
			// result, err := s.loadAccount(account)
			result := s.loadAccount(account)
			// result, code, err := s.loadAccount(account)
			// if err != nil {
			// 	log.Print(err)
			// 	bn, err := s.conn.BlockNumber(s.ctx)
			// 	if err != nil {
			// 		log.Panic(err)
			// 	}
			// 	log.Printf("Resetting AccountState... BN: %d", bn)
			// 	s.resetScratchPad(int64(bn))
			// 	goto UPDATE_BN_AND_RETRY
			// }
			nonce := uint64(result.Nonce)
			balance := result.Balance.ToInt()
			codeHash := result.CodeHash

			s.scratchPadMutex.Lock()
			if _, ok := s.scratchPad[calcKey(BALANCE_KEY, account)]; !ok {
				s.scratchPad[calcKey(BALANCE_KEY, account)] = balance.Bytes()
			}
			if _, ok := s.scratchPad[calcKey(NONCE_KEY, account)]; !ok {
				s.scratchPad[calcKey(NONCE_KEY, account)] = big.NewInt(int64(nonce)).Bytes()
			}
			if _, ok := s.scratchPad[calcKey(CODE_KEY, account)]; !ok {
				s.scratchPad[calcKey(CODE_KEY, account)] = result.Code
			}
			if _, ok := s.scratchPad[calcKey(CODEHASH_KEY, account)]; !ok {
				s.scratchPad[calcKey(CODEHASH_KEY, account)] = codeHash.Bytes()
			}

			switch action {
			case GET_BALANCE:
				res = s.scratchPad[calcKey(BALANCE_KEY, account)]
			case GET_NONCE:
				res = s.scratchPad[calcKey(NONCE_KEY, account)]
			case GET_CODE:
				res = s.scratchPad[calcKey(CODE_KEY, account)]
			case GET_CODEHASH:
				res = s.scratchPad[calcKey(CODEHASH_KEY, account)]
			}
			s.scratchPadMutex.Unlock()
		}
		return res, nil

	} else {
		if val, ok := s.scratchPad[scratchpadKey]; ok {
			// if action == GET_STATE {
			// log.Printf("got state at [depth:%d stateID: %02x], acc[%s].%s=0x%02x", s.deriveCnt, s.stateID, account.Hex(), key.Hex(), val)
			// }
			return val, nil
		}
		return s.parent.get(account, action, key)
	}
}

func (s *OverlayState) getRootState() *OverlayState {
	tmpState := s
	for {
		if tmpState.parent == nil {
			return tmpState
		} else {
			tmpState = tmpState.parent
		}
	}
}

func (s *OverlayState) DeriveFromRoot() *OverlayState {
	return s.getRootState().Derive("from root")
}

type OverlayStateDB struct {
	ctx  context.Context
	ec   *rpc.Client
	conn *ethclient.Client
	// block     int
	refundGas uint64
	state     *OverlayState
}

func (db *OverlayStateDB) GetOverlayDepth() int64 {
	return db.state.deriveCnt
}

func NewOverlayStateDB(rpcClient *rpc.Client, blockNumber int, keyCacheFilePath string, batchSize int) (db *OverlayStateDB) {
	db = &OverlayStateDB{
		ctx:       context.Background(),
		ec:        rpcClient,
		conn:      ethclient.NewClient(rpcClient),
		refundGas: 0,
	}
	state := NewOverlayState(db.ctx, db.ec, int64(blockNumber), keyCacheFilePath, batchSize).Derive("protect underlying") // protect underlying state
	db.state = state
	return db
}

func (db *OverlayStateDB) InitState(bn *uint64) {
	tmpDB := db.state
	reason := "reset and protect underlying"
	for {
		if tmpDB.parent == nil {
			db.state = tmpDB
			db.DestroyState()
			db.state.shouldStop = make(chan bool)
			var blockNumber uint64
			if bn == nil {
				var err error
				blockNumber, err = db.conn.BlockNumber(db.ctx)
				if err != nil {
					golog.Errorf("getBlockNumber error: %v, retrying", err)
					time.Sleep(time.Second * 3)
					continue
				}
			} else {
				blockNumber = *bn
			}

			golog.Infof("Resetting Scratchpad... BN: %d", bn)
			db.state.resetScratchPad(int64(blockNumber))
			golog.Info(reason)
			// log.Printf("pre driveID: %d", db.state.deriveCnt)
			db.state = db.state.Derive(reason)
			// log.Printf("post driveID: %d", db.state.deriveCnt)
			break
		} else {
			// log.Printf("pop scratchPad from: %d", tmpDB.deriveCnt)
			tmpDB = tmpDB.Parent()
		}
	}
}

func (db *OverlayStateDB) CreateAccount(account common.Address) {}

func (db *OverlayStateDB) SubBalance(account common.Address, delta *big.Int) {
	bal, err := db.state.get(account, GET_BALANCE, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	balB := new(big.Int).SetBytes(bal)
	post := balB.Sub(balB, delta)
	db.state.scratchPad[calcKey(BALANCE_KEY, account)] = post.Bytes()
}

func (db *OverlayStateDB) AddBalance(account common.Address, delta *big.Int) {
	bal, err := db.state.get(account, GET_BALANCE, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	balB := new(big.Int).SetBytes(bal)
	post := balB.Add(balB, delta)
	db.state.scratchPad[calcKey(BALANCE_KEY, account)] = post.Bytes()
}

func (db *OverlayStateDB) InitFakeAccounts() {
	db.AddBalance(constant.FAKE_ACCOUNT_0, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	db.AddBalance(constant.FAKE_ACCOUNT_1, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	db.AddBalance(constant.FAKE_ACCOUNT_2, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	db.AddBalance(constant.FAKE_ACCOUNT_3, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
}

func (db *OverlayStateDB) GetBalance(account common.Address) *big.Int {
	bal, err := db.state.get(account, GET_BALANCE, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	balB := new(big.Int).SetBytes(bal)
	return balB
}

func (db *OverlayStateDB) SetBalance(account common.Address, balance *big.Int) {
	db.state.scratchPad[calcKey(BALANCE_KEY, account)] = balance.Bytes()
}

func (db *OverlayStateDB) GetNonce(account common.Address) uint64 {
	nonce, err := db.state.get(account, GET_NONCE, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	nonceB := new(big.Int).SetBytes(nonce)
	return nonceB.Uint64()
}
func (db *OverlayStateDB) SetNonce(account common.Address, nonce uint64) {
	db.state.scratchPad[calcKey(NONCE_KEY, account)] = big.NewInt(int64(nonce)).Bytes()
}

func (db *OverlayStateDB) GetCodeHash(account common.Address) common.Hash {
	codehash, err := db.state.get(account, GET_CODEHASH, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	return common.BytesToHash(codehash)
}

func (db *OverlayStateDB) SetCodeHash(account common.Address, codeHash common.Hash) {
	db.state.scratchPad[calcKey(CODEHASH_KEY, account)] = codeHash.Bytes()
	if account.Hex() != (common.Address{}).Hex() {
		// log.Printf("SetCodeHash[depth:%d]: acc: %s key: %s, codehash: %s", db.state.deriveCnt, account.Hex(), calcKey( CODEHASH_KEY).Hex(), codeHash.Hex())
	}
}

func (db *OverlayStateDB) GetCode(account common.Address) []byte {
	code, err := db.state.get(account, GET_CODE, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	return code
}

func (db *OverlayStateDB) SetCode(account common.Address, code []byte) {
	db.state.scratchPad[calcKey(CODE_KEY, account)] = code
}

func (db *OverlayStateDB) GetCodeSize(account common.Address) int {
	code, err := db.state.get(account, GET_CODE, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	return len(code)
}

func (db *OverlayStateDB) AddRefund(delta uint64) { db.refundGas += delta }
func (db *OverlayStateDB) SubRefund(delta uint64) { db.refundGas -= delta }
func (db *OverlayStateDB) GetRefund() uint64      { return db.refundGas }

func (db *OverlayStateDB) GetCommittedState(account common.Address, key common.Hash) common.Hash {
	val, err := db.state.get(account, GET_STATE, key)
	if err != nil {
		log.Panic(err)
	}
	return common.BytesToHash(val)
}

func (db *OverlayStateDB) GetState(account common.Address, key common.Hash) common.Hash {
	v := db.GetCommittedState(account, key)
	// log.Printf("[R depth:%d, stateID:%02x] Acc: %s K: %s V: %s", db.state.deriveCnt, db.state.stateID, account.Hex(), key.Hex(), v.Hex())
	// log.Printf("Fetched: %s [%s] = %s", account.Hex(), key.Hex(), v.Hex())
	return v
}

func (db *OverlayStateDB) SetState(account common.Address, key common.Hash, value common.Hash) {
	// log.Printf("[W depth:%d stateID:%02x] Acc: %s K: %s V: %s", db.state.deriveCnt, db.state.stateID, account.Hex(), key.Hex(), value.Hex())
	db.state.scratchPad[calcStateKey(account, key)] = value.Bytes()
}

func (db *OverlayStateDB) Suicide(account common.Address) bool {
	db.state.scratchPad[calcKey(SUICIDE_KEY, account)] = []byte{0x01}
	return true
}

func (db *OverlayStateDB) HasSuicided(account common.Address) bool {
	if val, ok := db.state.scratchPad[calcKey(SUICIDE_KEY, account)]; ok {
		return bytes.Equal(val, []byte{0x01})
	}
	return false
}

func (db *OverlayStateDB) Exist(account common.Address) bool {
	return !db.Empty(account)
}

func (db *OverlayStateDB) Empty(account common.Address) bool {
	code := db.GetCode(account)
	nonce := db.GetNonce(account)
	balance := db.GetBalance(account)
	if len(code) == 0 && nonce == 0 && balance.Sign() == 0 {
		return true
	}
	return false
}

func (db *OverlayStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
}

func (db *OverlayStateDB) AddressInAccessList(addr common.Address) bool { return true }

func (db *OverlayStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return true, true
}

func (db *OverlayStateDB) AddAddressToAccessList(addr common.Address) { return }

func (db *OverlayStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) { return }

func (db *OverlayStateDB) RevertToSnapshot(revisionID int) {
	tmpState := db.state.Parent()
	close(db.state.shoudRevertSnapshot)
	golog.Infof("Rollbacking... revision: %d, currentID: %d", revisionID, tmpState.deriveCnt)
	for {
		if tmpState.deriveCnt+1 == int64(revisionID) {
			db.state = tmpState
			break
		} else {
			tmpState = tmpState.Parent()
		}
	}
}

func (db *OverlayStateDB) Snapshot() int {
	newOverlayState := db.state.Derive("snapshot")
	newOverlayState.shoudRevertSnapshot = make(chan bool)
	db.state = newOverlayState
	revisionID := int(newOverlayState.deriveCnt)
	return revisionID
}

func (db *OverlayStateDB) MergeTo(revisionID int) {
	currState, parentState := db.state, db.state.parent
	golog.Infof("Merging... target revisionID: %d, currentID: %d", revisionID, currState.deriveCnt)
	for {
		if currState.deriveCnt == int64(revisionID) {
			db.state = currState
			break
		}
		for k, v := range currState.scratchPad {
			parentState.scratchPad[k] = v
		}
		currState, parentState = parentState, parentState.parent
	}
}

func (db *OverlayStateDB) Clone() *OverlayStateDB {
	cpy := &OverlayStateDB{
		ctx:  db.ctx,
		ec:   db.ec,
		conn: db.conn,
		// block:     db.block,
		refundGas: 0,
		state:     db.state.Derive("clone"),
	}
	cpy.state.shouldStop = make(chan bool)
	return cpy
}

func (db *OverlayStateDB) DestroyState() {
	close(db.state.shouldStop)
}

func (db *OverlayStateDB) CloneFromRoot() *OverlayStateDB {
	cpy := &OverlayStateDB{
		ctx:  db.ctx,
		ec:   db.ec,
		conn: db.conn,
		// block:     db.block,
		refundGas: 0,
		state:     db.state.DeriveFromRoot(),
	}
	return cpy
}

func (db *OverlayStateDB) CacheSize() (size int) {
	root := db.state.getRootState()
	root.scratchPadMutex.RLock()
	defer root.scratchPadMutex.RUnlock()
	for k, v := range root.scratchPad {
		size += (len(k) + len(v))
	}
	return size
}

func (db *OverlayStateDB) RPCRequestCount() (cnt int64) {
	return db.state.getRootState().rpcCnt
}

func (db *OverlayStateDB) StateBlockNumber() (cnt int64) {
	return db.state.getRootState().bn
}

func (db *OverlayStateDB) AddLog(vLog *types.Log) {
	// spew.Dump(vLog)
	db.state.logs = append(db.state.logs, vLog)
}

func (db *OverlayStateDB) GetLogs(txHash common.Hash) []*types.Log {
	tmpStateDB := db.state
	logs := make([]*types.Log, 0)
	for {
		if tmpStateDB.txLogs[txHash] != nil {
			logs = append(tmpStateDB.txLogs[txHash], logs...)
		}
		if tmpStateDB.parent == nil {
			break
		}
		tmpStateDB = tmpStateDB.parent
	}
	return logs
}

func (db *OverlayStateDB) AddReceipt(txHash common.Hash, receipt *types.Receipt) {
	db.state.receipts[txHash] = receipt
}

func (db *OverlayStateDB) GetReceipt(txHash common.Hash) *types.Receipt {
	tmpStateDB := db.state
	for {
		if tmpStateDB.parent == nil {
			return nil
		}
		if receipt, ok := tmpStateDB.receipts[txHash]; ok {
			receipt.Logs = db.GetLogs(txHash)
			return receipt
		}
		tmpStateDB = tmpStateDB.parent
	}
}

func (db *OverlayStateDB) AddPreimage(common.Hash, []byte) {}

func (db *OverlayStateDB) ForEachStorage(account common.Address, callback func(common.Hash, common.Hash) bool) error {
	return nil
}

func (db *OverlayStateDB) StartLogCollection(txHash, blockHash common.Hash) {
	db.state.currentTxHash = txHash
	db.state.currentBlockHash = blockHash
	db.state.logs = make([]*types.Log, 0)
}

func (db *OverlayStateDB) FinishLogCollection() {
	for i := range db.state.logs {
		db.state.logs[i].BlockHash = db.state.currentBlockHash
		db.state.logs[i].TxHash = db.state.currentTxHash
	}
	db.state.txLogs[db.state.currentTxHash] = db.state.logs
}

func (db *OverlayStateDB) SetBatchSize(batchSize int) {
	db.state.getRootState().batchSize = batchSize
}
