package apestate

import (
	"bytes"
	"context"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type OverlayState struct {
	ctx                             context.Context
	ec                              *rpc.Client
	conn                            *ethclient.Client
	parent                          *OverlayState
	bn                              int64
	lastBN                          int64
	scratchPadMutex                 *sync.RWMutex
	scratchPad                      map[common.Hash][]byte
	logs                            []*types.Log
	txLogs                          map[common.Hash][]*types.Log
	receipts                        map[common.Hash]*types.Receipt
	currentTxHash, currentBlockHash common.Hash
	deriveCnt                       int64
	rpcCnt                          int64
}

func NewOverlayState(ctx context.Context, ec *rpc.Client, bn int64) *OverlayState {
	return &OverlayState{
		ctx:             ctx,
		ec:              ec,
		conn:            ethclient.NewClient(ec),
		parent:          nil,
		bn:              bn,
		scratchPadMutex: &sync.RWMutex{},
		scratchPad:      make(map[common.Hash][]byte),
		txLogs:          make(map[common.Hash][]*types.Log),
		logs:            make([]*types.Log, 0),
		receipts:        make(map[common.Hash]*types.Receipt),
		deriveCnt:       0,
	}
}

func (s *OverlayState) Derive(reason string) *OverlayState {
	// log.Printf("derive from: %s, depth: %d", reason, s.deriveCnt+1)
	return &OverlayState{
		// ctx:        s.ctx,
		// ec:         s.ec,
		// conn:       ethclient.NewClient(s.ec),
		// bn:         s.bn,
		parent:     s,
		scratchPad: make(map[common.Hash][]byte),
		txLogs:     make(map[common.Hash][]*types.Log),
		logs:       make([]*types.Log, 0),
		receipts:   make(map[common.Hash]*types.Receipt),
		deriveCnt:  s.deriveCnt + 1,
	}
}

func (s *OverlayState) Pop() *OverlayState {
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

	var result AccountResult
	var code hexutil.Bytes
	// fmt.Printf("loadAccount[%s] bigcache missed\r", account.Hex())
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

func (s *OverlayState) loadState(account common.Address, key common.Hash) (common.Hash, error) {
	storage, err := s.conn.StorageAt(s.ctx, account, key, big.NewInt(s.bn))
	if err != nil {
		return common.Hash{}, err
	}
	value := common.BytesToHash(storage)
	return value, nil
}

func calcKey(account common.Address, key common.Hash) common.Hash {
	return crypto.Keccak256Hash(account.Bytes(), key.Bytes())
}

func calcStateKey(account common.Address, key common.Hash) common.Hash {
	getStateKey := calcKey(account, STATE_KEY)
	stateKey := crypto.Keccak256Hash(getStateKey.Bytes(), key.Bytes())
	return stateKey
}

func (s *OverlayState) get(account common.Address, action RequestType, key common.Hash) ([]byte, error) {
	if s.parent == nil && s.bn != s.lastBN {
		log.Printf("State BN: %d", s.bn)
		s.lastBN = s.bn
	}
	var scratchpadKey common.Hash
	switch action {
	case GET_BALANCE:
		scratchpadKey = calcKey(account, BALANCE_KEY)
	case GET_NONCE:
		scratchpadKey = calcKey(account, NONCE_KEY)
	case GET_CODE:
		scratchpadKey = calcKey(account, CODE_KEY)
	case GET_CODEHASH:
		scratchpadKey = calcKey(account, CODEHASH_KEY)
	case GET_STATE:
		scratchpadKey = calcStateKey(account, key)
	}

	if s.parent == nil {
		s.scratchPadMutex.Lock()
		defer s.scratchPadMutex.Unlock()
		if val, ok := s.scratchPad[scratchpadKey]; ok {
			// s.scratchPadMutex.RUnlock()
			return val, nil
		}
		// s.scratchPadMutex.RUnlock()

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
				s.scratchPad = make(map[common.Hash][]byte)
				goto UPDATE_BN_AND_RETRY
			}
			// s.scratchPadMutex.Lock()
			s.scratchPad[scratchpadKey] = result.Bytes()
			res = s.scratchPad[scratchpadKey]
			// s.scratchPadMutex.Unlock()

		case GET_BALANCE, GET_NONCE, GET_CODE, GET_CODEHASH:
			result, code, err := s.loadAccount(account)
			if err != nil {
				log.Print(err)
				bn, err := s.conn.BlockNumber(s.ctx)
				if err != nil {
					log.Panic(err)
				}
				s.bn = int64(bn)
				log.Printf("Resetting AccountState... BN: %d", bn)
				s.scratchPad = make(map[common.Hash][]byte)
				goto UPDATE_BN_AND_RETRY
			}
			nonce := uint64(result.Nonce)
			balance := result.Balance.ToInt()
			codeHash := result.CodeHash

			// s.scratchPadMutex.Lock()
			if _, ok := s.scratchPad[calcKey(account, BALANCE_KEY)]; !ok {
				s.scratchPad[calcKey(account, BALANCE_KEY)] = balance.Bytes()
			}
			if _, ok := s.scratchPad[calcKey(account, NONCE_KEY)]; !ok {
				s.scratchPad[calcKey(account, NONCE_KEY)] = big.NewInt(int64(nonce)).Bytes()
			}
			if _, ok := s.scratchPad[calcKey(account, CODE_KEY)]; !ok {
				s.scratchPad[calcKey(account, CODE_KEY)] = code
			}
			if _, ok := s.scratchPad[calcKey(account, CODEHASH_KEY)]; !ok {
				s.scratchPad[calcKey(account, CODEHASH_KEY)] = codeHash[:]
			}

			switch action {
			case GET_BALANCE:
				res = s.scratchPad[calcKey(account, BALANCE_KEY)]
			case GET_NONCE:
				res = s.scratchPad[calcKey(account, NONCE_KEY)]
			case GET_CODE:
				res = s.scratchPad[calcKey(account, CODE_KEY)]
			case GET_CODEHASH:
				res = s.scratchPad[calcKey(account, CODEHASH_KEY)]
			}
			// s.scratchPadMutex.Unlock()
		}
		s.rpcCnt++
		return res, nil

	} else {
		if val, ok := s.scratchPad[scratchpadKey]; ok {
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
			tmpState = tmpState.Pop()
		}
	}
}

type OverlayStateDB struct {
	ctx       context.Context
	ec        *rpc.Client
	conn      *ethclient.Client
	block     int
	refundGas uint64
	state     *OverlayState
}

func (db *OverlayStateDB) GetOverlayDepth() int64 {
	return db.state.deriveCnt
}

func NewOverlayStateDB(rpcClient *rpc.Client, blockNumber int) (db *OverlayStateDB) {
	db = &OverlayStateDB{
		ctx:       context.Background(),
		ec:        rpcClient,
		conn:      ethclient.NewClient(rpcClient),
		block:     blockNumber,
		refundGas: 0,
	}
	state := NewOverlayState(db.ctx, db.ec, int64(db.block)).Derive("protect underlying") // protect underlying state
	db.state = state
	return db
}

func (db *OverlayStateDB) CloseCache() {
	tmpDB := db.state
	for {
		if tmpDB.parent == nil {
			db.state = tmpDB
			break
		} else {
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
	db.state.scratchPad[calcKey(account, BALANCE_KEY)] = post.Bytes()
}

func (db *OverlayStateDB) AddBalance(account common.Address, delta *big.Int) {
	bal, err := db.state.get(account, GET_BALANCE, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	balB := new(big.Int).SetBytes(bal)
	post := balB.Add(balB, delta)
	db.state.scratchPad[calcKey(account, BALANCE_KEY)] = post.Bytes()
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
	db.state.scratchPad[calcKey(account, BALANCE_KEY)] = balance.Bytes()
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
	db.state.scratchPad[calcKey(account, NONCE_KEY)] = big.NewInt(int64(nonce)).Bytes()
}

func (db *OverlayStateDB) GetCodeHash(account common.Address) common.Hash {
	codehash, err := db.state.get(account, GET_CODEHASH, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	return common.BytesToHash(codehash)
}

func (db *OverlayStateDB) SetCodeHash(account common.Address, codeHash common.Hash) {
	db.state.scratchPad[calcKey(account, CODEHASH_KEY)] = codeHash.Bytes()
	log.Printf("SetCodeHash[depth:%d]: acc: %s key: %s, codehash: %s", db.state.deriveCnt, account.Hex(), calcKey(account, CODEHASH_KEY).Hex(), codeHash.Hex())
}

func (db *OverlayStateDB) GetCode(account common.Address) []byte {
	code, err := db.state.get(account, GET_CODE, common.Hash{})
	if err != nil {
		log.Panic(err)
	}
	return code
}

func (db *OverlayStateDB) SetCode(account common.Address, code []byte) {
	db.state.scratchPad[calcKey(account, CODE_KEY)] = code
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
	// log.Printf("[R] Acc: %s K: %s V: %s", account.Hex(), key.Hex(), v.Hex())
	// log.Printf("Fetched: %s [%s] = %s", account.Hex(), key.Hex(), v.Hex())
	return v
}

func (db *OverlayStateDB) SetState(account common.Address, key common.Hash, value common.Hash) {
	// log.Printf("[W] Acc: %s K: %s V: %s", account.Hex(), key.Hex(), value.Hex())
	db.state.scratchPad[calcStateKey(account, key)] = value.Bytes()
}

func (db *OverlayStateDB) Suicide(account common.Address) bool {
	db.state.scratchPad[calcKey(account, SUICIDE_KEY)] = []byte{0x01}
	return true
}

func (db *OverlayStateDB) HasSuicided(account common.Address) bool {
	if val, ok := db.state.scratchPad[calcKey(account, SUICIDE_KEY)]; ok {
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
	log.Printf("Reverting... revision: %d, currentID: %d", revisionID, tmpState.deriveCnt)
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
		ctx:       db.ctx,
		ec:        db.ec,
		conn:      db.conn,
		block:     db.block,
		refundGas: 0,
		state:     db.state.Derive("clone"),
	}
	return cpy
}

func (db *OverlayStateDB) CloneFromRoot() *OverlayStateDB {
	cpy := &OverlayStateDB{
		ctx:       db.ctx,
		ec:        db.ec,
		conn:      db.conn,
		block:     db.block,
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
		tmpStateDB = tmpStateDB.Pop()
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
