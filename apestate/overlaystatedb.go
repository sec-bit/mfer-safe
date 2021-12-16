package apestate

import (
	"bytes"
	"context"
	"log"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/dynm/ape-safer/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
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

	logs                            []*types.Log
	txLogs                          map[common.Hash][]*types.Log
	receipts                        map[common.Hash]*types.Receipt
	currentTxHash, currentBlockHash common.Hash
	deriveCnt                       int64
	rpcCnt                          int64
	reqChan                         chan chan StorageReq
	loadAccountMutex                *sync.Mutex

	stateID uint64
}

func NewOverlayState(ctx context.Context, ec *rpc.Client, bn int64) *OverlayState {
	state := &OverlayState{
		ctx:             ctx,
		ec:              ec,
		conn:            ethclient.NewClient(ec),
		parent:          nil,
		bn:              bn,
		scratchPadMutex: &sync.RWMutex{},
		scratchPad:      make(map[string][]byte),

		txLogs:           make(map[common.Hash][]*types.Log),
		logs:             make([]*types.Log, 0),
		receipts:         make(map[common.Hash]*types.Receipt),
		deriveCnt:        0,
		reqChan:          make(chan chan StorageReq, 500),
		loadAccountMutex: &sync.Mutex{},
	}
	go state.timeSlot()
	return state
}

func (s *OverlayState) Derive(reason string) *OverlayState {
	// log.Printf("derive from: %s, depth: %d", reason, s.deriveCnt+1)
	return &OverlayState{
		parent:     s,
		scratchPad: make(map[string][]byte),
		txLogs:     make(map[common.Hash][]*types.Log),
		logs:       make([]*types.Log, 0),
		receipts:   make(map[common.Hash]*types.Receipt),
		deriveCnt:  s.deriveCnt + 1,

		stateID: rand.Uint64(),
	}
}

func (s *OverlayState) Pop() *OverlayState {
	s.scratchPad = make(map[string][]byte)
	return s.parent
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

func (s *OverlayState) loadAccount(account common.Address) (*AccountResult, []byte, error) {
	// s.loadAccountMutex.Lock()
	// defer s.loadAccountMutex.Unlock()
	var result AccountResult
	var code hexutil.Bytes
	rpcTries := 0
	blockNumber := hexutil.EncodeBig(big.NewInt(int64(s.bn)))

	for {
		err := s.ec.CallContext(s.ctx, &result, "eth_getProof", account, []string{}, blockNumber)
		if err != nil {
			rpcTries++
			if rpcTries > 5 {
				return nil, nil, err
			} else {
				log.Print("retrying getProof")
				time.Sleep(100 * time.Millisecond)
				continue
			}
		} else {
			rpcTries = 0
			break
		}
	}

	for {
		err := s.ec.CallContext(s.ctx, &code, "eth_getCode", account, blockNumber)
		if err != nil {
			rpcTries++
			if rpcTries > 5 {
				return nil, nil, err
			} else {
				log.Print("retrying getCode")
				time.Sleep(100 * time.Millisecond)
				continue
			}
		} else {
			rpcTries = 0
			break
		}
	}
	return &result, code, nil
}

func (s *OverlayState) loadStateBatchRPC(storageReqs []*StorageReq) error {
	// TODO: dedup

	s.rpcCnt++
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
	if err := s.ec.BatchCallContext(s.ctx, reqs); err != nil {
		return err
	}
	for i := range storageReqs {
		storageReqs[i].Value = values[i]
		// log.Printf("ReqHash: %s", storageReqs[i].Hash().Hex())
		// log.Printf("Batch Reqs: [%s].%s=%s", storageReqs[i].Address.Hex(), storageReqs[i].Key.Hex(), storageReqs[i].Value.Hex())
	}
	return nil
}

func (s *OverlayState) loadStateRPC(account common.Address, key common.Hash) (common.Hash, error) {
	s.rpcCnt++
	storage, err := s.conn.StorageAt(s.ctx, account, key, big.NewInt(s.bn))
	if err != nil {
		return common.Hash{}, err
	}
	value := common.BytesToHash(storage)
	return value, nil
}

func (s *OverlayState) timeSlot() {
	ticker := time.NewTicker(time.Millisecond * 3)
	for {
		reqLen := len(s.reqChan)
		if reqLen > 0 {
			// log.Printf("Request Len: %d", reqLen)
		}
		select {
		case <-ticker.C:
			reqPending := make([]*StorageReq, reqLen)
			reqChanPending := make([]chan StorageReq, reqLen)
			for i := 0; i < reqLen; i++ {
				req := <-s.reqChan
				storageReq := <-req
				reqPending[i] = &storageReq
				reqChanPending[i] = req
			}
			if reqLen == 0 {
				continue
			}
			err := s.loadStateBatchRPC(reqPending)
			if err != nil {
				log.Panic("loadStateBatch, err: ", err)
			}
			for i := 0; i < reqLen; i++ {
				req := reqChanPending[i]
				req <- *reqPending[i]
				close(req)
			}

		}
	}
}

func (s *OverlayState) loadState(account common.Address, key common.Hash) (common.Hash, error) {
	retChan := make(chan StorageReq)
	s.reqChan <- retChan
	retChan <- StorageReq{Address: account, Key: key}
	result := <-retChan
	// spew.Dump(result)
	return result.Value, result.Error
}

func calcKey(op common.Hash, account common.Address) string {
	return string(append(op.Bytes(), account.Bytes()...))
}

func calcStateKey(account common.Address, key common.Hash) string {
	getStateKey := calcKey(STATE_KEY, account)
	stateKey := getStateKey + string(key.Bytes())
	return stateKey
}

func (s *OverlayState) resetScratchPad() {
	s.scratchPadMutex.Lock()
	log.Printf("[reset scratchpad] lock scratchPad")

	reqs := make([]*StorageReq, 0)
	for key := range s.scratchPad {
		keyBytes := []byte(key)
		if len(keyBytes) == 32+20+32 && common.BytesToHash(keyBytes[:32]) == STATE_KEY {
			acc := common.BytesToAddress(keyBytes[32 : 32+20])
			key := common.BytesToHash(keyBytes[32+20:])
			reqs = append(reqs, &StorageReq{Address: acc, Key: key})
		}
	}
	err := s.loadStateBatchRPC(reqs)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("[reset scratchpad] sweeping scratchPad")
	s.scratchPad = make(map[string][]byte)

	for _, result := range reqs {
		stateKey := calcStateKey(result.Address, result.Key)
		s.scratchPad[stateKey] = result.Value[:]
		// log.Printf("resetted: [%s].%s=0x%02x\nstateKey: 0x%02x", result.Address.Hex(), result.Key.Hex(), s.scratchPad[stateKey], stateKey)
	}

	log.Printf("[reset scratchpad] pre fetch done, slot num: %d", len(s.scratchPad))

	s.scratchPadMutex.Unlock()
	log.Printf("[reset scratchpad] unlock scratchPad")

}

func (s *OverlayState) get(account common.Address, action RequestType, key common.Hash) ([]byte, error) {
	if s.parent == nil && s.bn != s.lastBN {
		log.Printf("State BN: %d", s.bn)
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
		s.scratchPadMutex.Lock()
		if val, ok := s.scratchPad[scratchpadKey]; ok {
			if action == GET_STATE {
				// log.Printf("got state at root, acc[%s].%s=0x%02x\nstateKey: %02x", account.Hex(), key.Hex(), val, scratchpadKey)
			}
			s.scratchPadMutex.Unlock()
			return val, nil
		}
		s.scratchPadMutex.Unlock()

		var res []byte
	UPDATE_BN_AND_RETRY:
		switch action {
		case GET_STATE:
			result, err := s.loadState(account, key)
			if err != nil {
				log.Print(err)
				bn, err := s.conn.BlockNumber(s.ctx)
				if err != nil {
					log.Panic(err)
				}
				s.bn = int64(bn)
				log.Printf("Resetting State... BN: %d", bn)
				s.resetScratchPad()
				goto UPDATE_BN_AND_RETRY
			}

			s.scratchPadMutex.Lock()
			s.scratchPad[scratchpadKey] = result.Bytes()
			// log.Printf("underlying get state, acc[%s].%s=%s", account.Hex(), key.Hex(), result.Hex())
			s.scratchPadMutex.Unlock()
			res = result.Bytes()

		case GET_BALANCE, GET_NONCE, GET_CODE, GET_CODEHASH:
			// log.Printf("underlying get account: %s", account.Hex())
			result, code, err := s.loadAccount(account)
			if err != nil {
				log.Print(err)
				bn, err := s.conn.BlockNumber(s.ctx)
				if err != nil {
					log.Panic(err)
				}
				s.bn = int64(bn)
				log.Printf("Resetting AccountState... BN: %d", bn)
				s.resetScratchPad()
				goto UPDATE_BN_AND_RETRY
			}
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
				s.scratchPad[calcKey(CODE_KEY, account)] = code
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
			if action == GET_STATE {
				// log.Printf("got state at [depth:%d stateID: %02x], acc[%s].%s=0x%02x", s.deriveCnt, s.stateID, account.Hex(), key.Hex(), val)
			}
			return val, nil
		}
		return s.parent.get(account, action, key)
	}
}

func (s *OverlayState) DeriveFromRoot() *OverlayState {
	tmpState := s
	for {
		if tmpState.parent == nil {
			return tmpState.Derive("from root")
		} else {
			tmpState = tmpState.parent
		}
	}
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

func NewOverlayStateDB(rpcClient *rpc.Client, blockNumber int) (db *OverlayStateDB) {
	db = &OverlayStateDB{
		ctx:  context.Background(),
		ec:   rpcClient,
		conn: ethclient.NewClient(rpcClient),
		// block:     blockNumber,
		refundGas: 0,
	}
	state := NewOverlayState(db.ctx, db.ec, int64(blockNumber)).Derive("protect underlying") // protect underlying state
	db.state = state
	return db
}

func (db *OverlayStateDB) CloseCache() {
	tmpDB := db.state
	reason := "reset and protect underlying"
	for {
		if tmpDB.parent == nil {
			db.state = tmpDB
			db.state.resetScratchPad()
			log.Print(reason)
			// log.Printf("pre driveID: %d", db.state.deriveCnt)
			db.state = db.state.Derive(reason)
			// log.Printf("post driveID: %d", db.state.deriveCnt)
			break
		} else {
			// log.Printf("pop scratchPad from: %d", tmpDB.deriveCnt)
			tmpDB = tmpDB.Pop()
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
	tmpState := db.state.Pop()
	log.Printf("Rollbacking... revision: %d, currentID: %d", revisionID, tmpState.deriveCnt)
	for {
		if tmpState.deriveCnt+1 == int64(revisionID) {
			db.state = tmpState
			break
		} else {
			tmpState = tmpState.Pop()
		}
	}
}

func (db *OverlayStateDB) Snapshot() int {
	newOverlayState := db.state.Derive("snapshot")
	db.state = newOverlayState
	revisionID := int(newOverlayState.deriveCnt)
	// log.Printf("TakeSnapshot: %d", revisionID)
	return revisionID
}

func (db *OverlayStateDB) MergeTo(revisionID int) {
	currState, parentState := db.state, db.state.parent
	log.Printf("Merging... target revisionID: %d, currentID: %d", revisionID, currState.deriveCnt)
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
	return cpy
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
	root := db.state.DeriveFromRoot().parent
	root.scratchPadMutex.RLock()
	defer root.scratchPadMutex.RUnlock()
	for k, v := range root.scratchPad {
		size += (len(k) + len(v))
	}
	return size
}

func (db *OverlayStateDB) RPCRequestCount() (cnt int64) {
	return db.state.DeriveFromRoot().parent.rpcCnt
}

func (db *OverlayStateDB) StateBlockNumber() (cnt int64) {
	return db.state.DeriveFromRoot().parent.bn
}

func (db *OverlayStateDB) AddLog(vLog *types.Log) {
	// spew.Dump(vLog)
	db.state.logs = append(db.state.logs, vLog)
}

func (db *OverlayStateDB) GetLogs(txHash common.Hash) []*types.Log {
	return db.state.txLogs[txHash]
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
