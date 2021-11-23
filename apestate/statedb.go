package apestate

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type Account struct {
	ctx      context.Context
	conn     *ethclient.Client
	Address  common.Address
	Nonce    uint64
	Balance  *big.Int
	Code     []byte
	CodeHash common.Hash
	State    map[common.Hash]common.Hash

	accCache *Cache
	suicided bool
}

func (acc *Account) Copy() *Account {
	accCpy := &Account{
		ctx:      acc.ctx,
		conn:     acc.conn,
		Address:  acc.Address,
		Nonce:    acc.Nonce,
		Balance:  new(big.Int).Set(acc.Balance),
		Code:     acc.Code,
		CodeHash: acc.CodeHash,
		suicided: acc.suicided,
	}
	accCpy.State = make(map[common.Hash]common.Hash)
	for k, v := range acc.State {
		accCpy.State[k] = v
	}
	return accCpy
}

func (acc *Account) fetchStorage(key common.Hash, bn int) (common.Hash, error) {
	var blockNumber *big.Int
	if bn != -1 {
		blockNumber = big.NewInt(int64(bn))
	}

	if v, err := acc.accCache.GetState(acc.Address, key); err == nil {
		return v, nil
	} else {
		// log.Printf("cache miss: acc[%s] key=%s", acc.Address.Hex(), key.Hex())
	}

	storage, err := acc.conn.StorageAt(acc.ctx, acc.Address, key, blockNumber)
	if err != nil {
		return common.Hash{}, err
	}
	value := common.BytesToHash(storage)

	// log.Printf("caching acc[%s] key: %s value: %s", acc.Address.Hex(), key.Hex(), value.Hex())
	acc.accCache.SetState(acc.Address, key, value)

	return value, nil
}

func (acc *Account) loadState(key common.Hash, bn int) (common.Hash, error) {
	// start := time.Now()
	// defer func() {
	// 	log.Printf("loadState consumes: %v", time.Since(start))
	// }()
	if v, ok := acc.State[key]; !ok {
		v, err := acc.fetchStorage(key, bn)
		if err != nil {
			return v, err
		}
		acc.State[key] = v
		return v, nil
	} else {
		return v, nil
	}
}

type CACHETYPE int

const (
	ACCOUNT CACHETYPE = iota
	CODE
	STATE
)

type Cache struct {
	b *bigcache.BigCache
}

func (c *Cache) GetAccount(acc common.Address) (*RPCAccountCache, error) {
	key := append(acc.Bytes(), byte(ACCOUNT))
	ret, err := c.b.Get(string(key))
	if err != nil {
		return nil, err
	}

	var rpcAccCache RPCAccountCache
	err = json.Unmarshal(ret, &rpcAccCache)
	if err != nil {
		log.Printf("GetAccount error decoding: %v", err)
		return nil, err
	}

	codekey := append(rpcAccCache.AccResult.CodeHash[:], byte(CODE))
	code, err := c.b.Get(string(codekey))
	if err != nil {
		return nil, err
	}
	rpcAccCache.Code = hexutil.Bytes(code)
	return &rpcAccCache, nil
}

func (c *Cache) SetAccount(acc common.Address, accCache *RPCAccountCache) {
	key := append(acc.Bytes(), byte(ACCOUNT))
	code := accCache.Code
	accCache.Code = nil
	enc, err := json.Marshal(accCache)
	// log.Printf("encoded: %s", enc)
	if err != nil {
		log.Printf("SetAccount error encoding: %v", err)
		return
	}

	c.b.Set(string(key), enc)
	codekey := append(accCache.AccResult.CodeHash[:], byte(CODE))
	c.b.Set(string(codekey), code)
	// return
}

func (c *Cache) GetState(acc common.Address, key common.Hash) (common.Hash, error) {
	ckey := append(acc.Bytes(), byte(STATE))
	ckey = append(ckey, key.Bytes()...)

	// log.Printf("GetState acc: %s key: %s", acc.Hex(), key.Hex())
	// if c.b == nil {
	// 	log.Panic("c.b is nil")
	// }

	ret, err := c.b.Get(string(ckey))
	if err != nil {
		return common.Hash{}, err
	}
	var val common.Hash
	err = json.Unmarshal(ret, &val)
	if err != nil {
		log.Printf("GetState error decoding: %v", err)
		return common.Hash{}, err
	}
	return val, nil
}

func (c *Cache) SetState(acc common.Address, key, val common.Hash) {
	ckey := append(acc.Bytes(), byte(STATE))
	ckey = append(ckey, key.Bytes()...)

	obj, err := json.Marshal(val)
	if err != nil {
		log.Printf("SetState error encoding: %v", err)
		return
	}

	c.b.Set(string(ckey), obj)
	return
}

type RPCAccountCache struct {
	AccResult *AccountResult
	Code      hexutil.Bytes
}

type ApeStateDB struct {
	ctx        context.Context
	ec         *rpc.Client
	conn       *ethclient.Client
	block      int
	accountMap map[common.Address]*Account
	refundGas  uint64

	snapshots    map[int]map[common.Address]*Account
	snapshotLog  map[int][]*types.Log
	snapshotID   int
	snapshotLock *sync.RWMutex

	logs                            []*types.Log
	txLogs                          map[common.Hash][]*types.Log
	currentTxHash, currentBlockHash common.Hash
	receipts                        map[common.Hash]*types.Receipt

	globalCache     *Cache
	globalCodeCache map[common.Hash][]byte
}

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

func NewApeStateDB(rpcClient *rpc.Client, blockNumber int) (db *ApeStateDB) {
	cfg := bigcache.DefaultConfig(5 * time.Minute)
	cfg.Verbose = false
	cache, _ := bigcache.NewBigCache(cfg)
	accCache := &Cache{b: cache}
	db = &ApeStateDB{
		ctx:        context.Background(),
		ec:         rpcClient,
		conn:       ethclient.NewClient(rpcClient),
		block:      blockNumber,
		accountMap: make(map[common.Address]*Account),

		snapshots:    make(map[int]map[common.Address]*Account),
		snapshotLog:  make(map[int][]*types.Log),
		snapshotLock: &sync.RWMutex{},

		txLogs:    make(map[common.Hash][]*types.Log),
		refundGas: 0,
		receipts:  make(map[common.Hash]*types.Receipt),

		globalCache:     accCache,
		globalCodeCache: make(map[common.Hash][]byte),
	}
	return
}

func (db *ApeStateDB) CloseCache() {
	db.globalCache.b.Close()
}

func (db *ApeStateDB) loadAccount(account common.Address) (*Account, error) {
	// start := time.Now()
	// defer func() {
	// 	log.Printf("loadAccount[%s] consumes: %v", account.Hex(), time.Since(start))
	// }()

	if acc, ok := db.accountMap[account]; ok {
		// log.Printf("loadAccount[%s] cache hitted", account.Hex())
		return acc, nil
	} else {
		// log.Printf("loadAccount[%s] cache missed", account.Hex())
		blockNumber := "latest"
		if db.block != -1 {
			blockNumber = hexutil.EncodeBig(big.NewInt(int64(db.block)))
		}

		var result AccountResult
		var code hexutil.Bytes

		foundCache := false
		if val, err := db.globalCache.GetAccount(account); err == nil {
			if val.AccResult != nil {
				result = *val.AccResult
				code = val.Code
				foundCache = true
			}
		}

		if !foundCache {
			// fmt.Printf("loadAccount[%s] bigcache missed\r", account.Hex())
			rpcTries := 0

			for {
				err := db.ec.CallContext(db.ctx, &result, "eth_getProof", account, []string{}, blockNumber)
				if err != nil {
					rpcTries++
					if rpcTries > 5 {
						return nil, err
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
				err := db.ec.CallContext(db.ctx, &code, "eth_getCode", account, blockNumber)
				if err != nil {
					rpcTries++
					if rpcTries > 5 {
						return nil, err
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
		}

		nonce := uint64(result.Nonce)
		balance := result.Balance.ToInt()
		codeHash := result.CodeHash

		acc = &Account{
			ctx:      db.ctx,
			conn:     db.conn,
			Address:  account,
			Nonce:    nonce,
			Balance:  balance,
			CodeHash: codeHash,
			Code:     code,
			State:    make(map[common.Hash]common.Hash),

			accCache: db.globalCache,
		}
		if acc.accCache == nil {
			log.Panic("accCache nil")
		}
		if acc.accCache.b == nil {
			log.Panic("accCache.b nil")
		}
		db.accountMap[account] = acc

		result.AccountProof = nil
		result.StorageProof = nil

		accCache := &RPCAccountCache{
			AccResult: &result,
			Code:      code,
		}
		db.globalCache.SetAccount(account, accCache)
		return acc, nil
	}

}

func (db *ApeStateDB) CreateAccount(account common.Address) {}

func (db *ApeStateDB) SubBalance(account common.Address, delta *big.Int) {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	acc.Balance.Sub(acc.Balance, delta)
}

func (db *ApeStateDB) AddBalance(account common.Address, delta *big.Int) {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	acc.Balance.Add(acc.Balance, delta)
}

func (db *ApeStateDB) GetBalance(account common.Address) *big.Int {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	return acc.Balance
}

func (db *ApeStateDB) SetBalance(account common.Address, balance *big.Int) {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	acc.Balance = balance
}

func (db *ApeStateDB) GetNonce(account common.Address) uint64 {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	return acc.Nonce
}
func (db *ApeStateDB) SetNonce(account common.Address, nonce uint64) {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	acc.Nonce = nonce
}

func (db *ApeStateDB) GetCodeHash(account common.Address) common.Hash {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	return acc.CodeHash
}

func (db *ApeStateDB) SetCodeHash(account common.Address, codeHash common.Hash) {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	acc.CodeHash = codeHash
}

func (db *ApeStateDB) GetCode(account common.Address) []byte {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	// log.Printf("GetCode [acc:%s], Code Size: %d", account.Hex(), len(acc.Code))
	return acc.Code
}

func (db *ApeStateDB) SetCode(account common.Address, code []byte) {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	codeHash := crypto.Keccak256Hash(code)
	acc.CodeHash = codeHash
	acc.Code = code
}

func (db *ApeStateDB) GetCodeSize(account common.Address) int {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	return len(acc.Code)
}

func (db *ApeStateDB) AddRefund(delta uint64) { db.refundGas += delta }
func (db *ApeStateDB) SubRefund(delta uint64) { db.refundGas -= delta }
func (db *ApeStateDB) GetRefund() uint64      { return db.refundGas }

func (db *ApeStateDB) GetCommittedState(account common.Address, key common.Hash) common.Hash {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	var v common.Hash
	cnt := 0
	for {
		acc.accCache = db.globalCache
		v, err = acc.loadState(key, db.block)
		if err == nil {
			break
		} else {
			log.Print("getCommittedState error: ", err)
			time.Sleep(time.Second)
		}
		if cnt > 3 {
			log.Panic("tried 3 times, give up")
		}
		cnt++
	}

	return v
}

func (db *ApeStateDB) GetState(account common.Address, key common.Hash) common.Hash {
	v := db.GetCommittedState(account, key)
	// log.Printf("[R] Acc: %s K: %s V: %s", account.Hex(), key.Hex(), v.Hex())
	// log.Printf("Fetched: %s [%s] = %s", account.Hex(), key.Hex(), v.Hex())
	return v
}

func (db *ApeStateDB) SetState(account common.Address, key common.Hash, value common.Hash) {
	// log.Printf("[W] Acc: %s K: %s V: %s", account.Hex(), key.Hex(), value.Hex())
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	acc.State[key] = value
}

func (db *ApeStateDB) Suicide(account common.Address) bool {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	acc.suicided = true
	acc.Balance = new(big.Int)
	return true
}

func (db *ApeStateDB) HasSuicided(account common.Address) bool {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	return acc.suicided
}

func (db *ApeStateDB) Exist(account common.Address) bool {
	return !db.Empty(account)
}

func (db *ApeStateDB) Empty(account common.Address) bool {
	acc, err := db.loadAccount(account)
	if err != nil {
		log.Panic(err)
	}
	if len(acc.Code) == 0 && acc.Nonce == 0 && acc.Balance.Sign() == 0 {
		return true
	}
	return false
}

func (db *ApeStateDB) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
}

func (db *ApeStateDB) AddressInAccessList(addr common.Address) bool { return false }

func (db *ApeStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return true, true
}

func (db *ApeStateDB) AddAddressToAccessList(addr common.Address) { return }

func (db *ApeStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) { return }

func (db *ApeStateDB) RevertToSnapshot(revisionID int) {
	db.snapshotLock.Lock()
	defer db.snapshotLock.Unlock()
	db.accountMap = db.snapshots[revisionID]
	db.logs = db.snapshotLog[revisionID]
	delete(db.snapshots, revisionID)
	delete(db.snapshotLog, revisionID)
}

func (db *ApeStateDB) Snapshot() int {
	snapshot := make(map[common.Address]*Account)
	for addr, acc := range db.accountMap {
		snapshot[addr] = acc.Copy()
	}
	db.snapshotLock.Lock()
	defer db.snapshotLock.Unlock()
	db.snapshots[db.snapshotID] = snapshot
	db.snapshotLog[db.snapshotID] = db.logs
	revisionID := db.snapshotID
	db.snapshotID++
	// log.Printf("snapshot  id: %d, snapshots cnt: %d", db.snapshotID-1, len(db.snapshots))
	return revisionID
}

func (db *ApeStateDB) Clone() *ApeStateDB {
	snapshot := make(map[common.Address]*Account)
	for addr, acc := range db.accountMap {
		snapshot[addr] = acc.Copy()
	}

	cpy := &ApeStateDB{
		ctx:        db.ctx,
		ec:         db.ec,
		conn:       db.conn,
		block:      db.block,
		accountMap: snapshot,

		snapshots:    make(map[int]map[common.Address]*Account),
		snapshotLog:  make(map[int][]*types.Log),
		snapshotLock: &sync.RWMutex{},

		txLogs:    make(map[common.Hash][]*types.Log),
		refundGas: 0,
		receipts:  make(map[common.Hash]*types.Receipt),

		globalCache: db.globalCache,
	}
	return cpy
}

func (db *ApeStateDB) CacheSize() (size int) {
	return db.globalCache.b.Capacity()
}

func (db *ApeStateDB) AddLog(vLog *types.Log) {
	// spew.Dump(vLog)
	db.logs = append(db.logs, vLog)
}

func (db *ApeStateDB) GetLogs(txHash common.Hash) []*types.Log {
	return db.txLogs[txHash]
}

func (db *ApeStateDB) AddReceipt(txHash common.Hash, receipt *types.Receipt) {
	db.receipts[txHash] = receipt
}

func (db *ApeStateDB) GetReceipt(txHash common.Hash) *types.Receipt {
	return db.receipts[txHash]
}

func (db *ApeStateDB) AddPreimage(common.Hash, []byte) {}

func (db *ApeStateDB) ForEachStorage(account common.Address, callback func(common.Hash, common.Hash) bool) error {
	return nil
}

func (db *ApeStateDB) StartLogCollection(txHash, blockHash common.Hash) {
	db.currentTxHash = txHash
	db.currentBlockHash = blockHash
	db.logs = make([]*types.Log, 0)
}

func (db *ApeStateDB) FinishLogCollection() {
	for i := range db.logs {
		db.logs[i].BlockHash = db.currentBlockHash
		db.logs[i].TxHash = db.currentTxHash
	}
	db.txLogs[db.currentTxHash] = db.logs
}
