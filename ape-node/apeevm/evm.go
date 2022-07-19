package apeevm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/dustin/go-humanize"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/kataras/golog"

	"github.com/dynm/ape-safer/apesigner"
	"github.com/dynm/ape-safer/apestate"
	"github.com/dynm/ape-safer/constant"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type ApeEVM struct {
	ctx        context.Context
	RpcClient  *rpc.Client
	Conn       *ethclient.Client
	SelfClient *rpc.Client
	SelfConn   *ethclient.Client

	StateDB             *apestate.OverlayStateDB
	keyCacheFilePath    string
	batchSize           int
	vmContext           vm.BlockContext
	gasPool             *core.GasPool
	chainConfig         *params.ChainConfig
	callMutex           *sync.RWMutex
	stateLock           *sync.RWMutex
	impersonatedAccount common.Address
	timeDelta           uint64
	blockNumberDelta    uint64
	tracer              vm.Tracer
}

func NewApeEVM(rawurl string, impersonatedAccount common.Address, keyCacheFilePath string, batchSize int) *ApeEVM {
	apeEVM := &ApeEVM{}
	splittedRawUrl := strings.Split(rawurl, "@")
	var specificBlock *int
	if len(splittedRawUrl) > 1 {
		bnStr := splittedRawUrl[len(splittedRawUrl)-1]
		bn, err := strconv.Atoi(bnStr)
		if err != nil {
			log.Panic(err)
		}
		specificBlock = &bn
		lastIndex := strings.LastIndex(rawurl, "@"+bnStr)
		rawurl = rawurl[:lastIndex]
	}
	ctx := context.Background()
DIAL:
	RpcClient, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		golog.Errorf("Dial [%s] error: [%v] retrying", rawurl, err)
		time.Sleep(time.Second * 3)
		goto DIAL
	}
	apeEVM.ctx = ctx
	apeEVM.RpcClient = RpcClient
	apeEVM.Conn = ethclient.NewClient(RpcClient)
	apeEVM.callMutex = &sync.RWMutex{}
	apeEVM.stateLock = &sync.RWMutex{}
	apeEVM.impersonatedAccount = impersonatedAccount
	apeEVM.keyCacheFilePath = keyCacheFilePath
	apeEVM.batchSize = batchSize
	err = apeEVM.Prepare(specificBlock)
	if err != nil {
		golog.Errorf("Prepare error: %v", err)
		time.Sleep(time.Second)
		goto DIAL
	}

	if specificBlock == nil {
		go apeEVM.updatePendingBN()
	} else {
		golog.Infof("Using specific block %d, auto update block context disabled", *specificBlock)
	}

	return apeEVM
}

func (a *ApeEVM) StateLock() {
	a.stateLock.Lock()
}

func (a *ApeEVM) StateUnlock() {
	a.stateLock.Unlock()
}

func (a *ApeEVM) GetLatestBlockHeader() *types.Header {
	var raw json.RawMessage
	err := a.RpcClient.CallContext(a.ctx, &raw, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		golog.Errorf("GetBlockHeader err: %v", err)
		return nil
	} else if len(raw) == 0 {
		golog.Errorf("GetBlockHeader: Block not found")
		return nil
	}
	// Decode header and transactions.
	var head types.Header
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil
	}

	return &head
}
func (a *ApeEVM) ResetState() {
	lastBlockHeader := a.GetLatestBlockHeader()
	if lastBlockHeader == nil {
		return
	}
	a.StateDB.InitState(nil)
	a.gasPool = new(core.GasPool)
	a.gasPool.AddGas(lastBlockHeader.GasLimit)
}

func (a *ApeEVM) ChainID() *big.Int {
	return a.chainConfig.ChainID
}

func (a *ApeEVM) Prepare(bn *int) error {
	a.chainConfig = core.DefaultGenesisBlock().Config
	chainID, err := a.Conn.ChainID(a.ctx)
	if err != nil {
		return err
	}
	a.chainConfig.ChainID = chainID

	//avoid invalid opcode: SHR
	a.chainConfig.ByzantiumBlock = big.NewInt(0)
	a.chainConfig.ConstantinopleBlock = big.NewInt(0)

	lastBlockHeader := a.GetLatestBlockHeader()
	if lastBlockHeader == nil {
		return fmt.Errorf("cannot get last block")
	}

	if a.StateDB == nil {
		var blockNumber int
		if bn == nil {
			blockNumber = int(lastBlockHeader.Number.Int64())
		} else {
			blockNumber = *bn
		}
		a.StateDB = apestate.NewOverlayStateDB(a.RpcClient, blockNumber, a.keyCacheFilePath, a.batchSize)
		a.StateDB.InitState(nil)
	}
	a.StateDB.InitFakeAccounts()

	getHash := func(bn uint64) common.Hash {
		blk, err := a.Conn.BlockByNumber(a.ctx, new(big.Int).SetUint64(bn))
		if err != nil {
			return common.Hash{}
		}
		return blk.Hash()
	}
	a.gasPool = new(core.GasPool)
	// a.gasPool.AddGas(lastBlockHeader.GasLimit)
	a.gasPool.AddGas(math.MaxUint64)
	a.vmContext = vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    common.HexToAddress("0xaabbccddaabbccddaabbccddaabbccddaabbccdd"),
		GetHash:     getHash,
		BaseFee:     big.NewInt(0),
		BlockNumber: big.NewInt(0),
		Time:        big.NewInt(0),
		Difficulty:  big.NewInt(0),
	}
	if bn == nil {
		a.setVMContext()
	}
	return nil
}

func (a *ApeEVM) GetChainConfig() params.ChainConfig {
	return *a.chainConfig
}

func (a *ApeEVM) SetTimeDelta(delta uint64) {
	a.timeDelta = delta
}

func (a *ApeEVM) SetBlockNumberDelta(delta uint64) {
	a.blockNumberDelta = delta
}

func (a *ApeEVM) setVMContext() {
	lastBlockHeader := a.GetLatestBlockHeader()
	if lastBlockHeader == nil {
		return
	}

	a.vmContext.BlockNumber.SetInt64(int64(lastBlockHeader.Number.Uint64() + 1 + a.blockNumberDelta))
	a.vmContext.Time.SetInt64(int64(lastBlockHeader.Time + a.timeDelta))
	a.vmContext.Difficulty.Set(lastBlockHeader.Difficulty)
	a.vmContext.GasLimit = lastBlockHeader.GasLimit
}

func (a *ApeEVM) SetVMContextByBlock(block *types.Block) {
	header := block.Header()
	a.vmContext.BlockNumber.SetInt64(int64(header.Number.Uint64()))
	a.vmContext.Time.SetInt64(int64(header.Time + a.timeDelta))
	a.vmContext.Difficulty.Set(header.Difficulty)
	a.vmContext.GasLimit = header.GasLimit
}

func (a *ApeEVM) GetVMContext() vm.BlockContext {
	return a.vmContext
}

func (a *ApeEVM) updatePendingBN() {
	headerChan := make(chan *types.Header)
	ticker5Sec := time.NewTicker(time.Second * 5)
	tickerCheckMissingTireNode := time.NewTicker(time.Second * 10)

	sub, err := a.Conn.SubscribeNewHead(a.ctx, headerChan)
	if err != nil {
		golog.Warnf("subscribe err: %v, use poll instead", err)
	} else {
		ticker5Sec.Stop()
		go func() {
			for {
				<-sub.Err()
				golog.Errorf("sub err=%v, resubscribing", err)
			RESUB:
				sub, err = a.Conn.SubscribeNewHead(a.ctx, headerChan)
				if err != nil {
					golog.Errorf("sub err=%v, retrying", err)
					time.Sleep(time.Second)
					goto RESUB
				}
			}
		}()

	}
	for {
		select {
		case <-tickerCheckMissingTireNode.C:
			stateHeight := a.StateDB.StateBlockNumber()
			golog.Infof("Checking if height@%d(0x%02x) is missing", stateHeight, stateHeight)
			balance, err := a.Conn.BalanceAt(a.ctx, common.HexToAddress("0x0000000000000000000000000000000000000000"), big.NewInt(stateHeight))
			if err != nil {
				golog.Error(err)
			}
			if err != nil && strings.Contains(err.Error(), "missing trie node") {
				golog.Warn("InitState (missing trie node)")
				// a.StateDB.InitState()
				a.SelfClient.Call(nil, "ape_reExecTxPool")
			} else if balance.Sign() == 0 { //some node will not tell us missing trie node
				golog.Warn("InitState (0x0000...0000 balance is zero)")
				// a.StateDB.InitState()
				a.SelfClient.Call(nil, "ape_reExecTxPool")
			}

		case <-ticker5Sec.C:
			a.setVMContext()
		case <-headerChan:
			a.setVMContext()
		}
		sizeStr := humanize.Bytes(uint64(a.StateDB.CacheSize()))
		golog.Infof("[Update] BN: %d, StateBlock: %d, Ts: %d, Diff: %d, GasLimit: %d, Cache: %s, RPCReq: %d",
			a.vmContext.BlockNumber, a.StateDB.StateBlockNumber(), a.vmContext.Time, a.vmContext.Difficulty, a.vmContext.GasLimit, sizeStr, a.StateDB.RPCRequestCount())
	}

}

var (
	rootHash  = crypto.Keccak256Hash([]byte("fake state root"))
	blockHash = crypto.Keccak256Hash([]byte("fake block hash"))
)

func (a *ApeEVM) SetTracer(t vm.Tracer) {
	a.tracer = t
}

func (a *ApeEVM) TxToMessage(tx *types.Transaction) types.Message {
	v, r, s := tx.RawSignatureValues()
	var signer types.Signer
	if v.Uint64() == 1 && bytes.Equal(s.Bytes(), constant.APESIGNER_S.Bytes()) && r != nil {
		signer = apesigner.NewSigner(a.ChainID().Int64())
	} else {
		signer = types.NewLondonSigner(a.ChainID())
	}
	msg, _ := tx.AsMessage(signer, nil)
	return msg
}

func (a *ApeEVM) WarmUpCache(txs types.Transactions, stateDB *apestate.OverlayStateDB) {
	wg := sync.WaitGroup{}
	txCh := make(chan *types.Transaction, 100)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(db *apestate.OverlayStateDB) {
			defer wg.Done()
			for tx := range txCh {
				msg := a.TxToMessage(tx)
				stateDB := db.CloneFromRoot()
				gp := new(core.GasPool)
				gp.AddGas(math.MaxUint64)
				// stateDB.(*apestate.OverlayStateDB).SetCodeHash(msg.From(), common.Hash{})
				txContext := core.NewEVMTxContext(msg)
				evm := vm.NewEVM(a.vmContext, txContext, stateDB, a.chainConfig, vm.Config{})
				stateDB.StartLogCollection(tx.Hash(), blockHash)
				core.ApplyMessage(evm, msg, gp)
			}
		}(stateDB)
	}
	for _, tx := range txs {
		txCh <- tx
	}
	close(txCh)
	wg.Wait()
}

func (a *ApeEVM) ExecuteTxs(txs types.Transactions, stateDB vm.StateDB, config *tracers.TraceConfig) (execResults []error) {
	execResults = make([]error, len(txs))
	var (
		gasUsed = uint64(0)
		txIndex = 0
	)
	var (
		tracer vm.Tracer
		err    error
	)
	for i, tx := range txs {
		txctx := &tracers.Context{
			TxHash:  tx.Hash(),
			TxIndex: i,
		}

		switch {
		case config != nil && config.Tracer != nil:
			// Define a meaningful timeout of a single transaction trace
			timeout := time.Second * 1
			if config.Timeout != nil {
				if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
					return nil
				}
			}
			// Constuct the JavaScript tracer to execute with
			if tracer, err = tracers.New(*config.Tracer, txctx); err != nil {
				return nil
			}
			// Handle timeouts and RPC cancellations
			deadlineCtx, cancel := context.WithTimeout(context.Background(), timeout)
			go func() {
				<-deadlineCtx.Done()
				if deadlineCtx.Err() == context.DeadlineExceeded {
					tracer.(*tracers.Tracer).Stop(errors.New("execution timeout"))
				}
			}()
			defer cancel()

		case config == nil:
			tracer, err = tracers.New("callTracer", txctx)
			if err != nil {
				log.Panic(err)
			}
		default:
			config.LogConfig.EnableMemory = true
			config.LogConfig.EnableReturnData = true
			config.LogConfig.DisableStorage = false
			tracer = vm.NewStructLogger(config.LogConfig)
		}
		msg := a.TxToMessage(tx)
		stateDB.(*apestate.OverlayStateDB).SetCodeHash(msg.From(), common.Hash{})
		// log.Printf("From: %s, To: %s, Nonce: %d, GasPrice: %d, Gas: %d, Hash: %s", msg.From(), msg.To(), msg.Nonce(), msg.GasPrice(), msg.Gas(), tx.Hash())

		txContext := core.NewEVMTxContext(msg)
		snapshot := stateDB.Snapshot()

		// a.vmContext.BlockNumber.Add(a.vmContext.BlockNumber, big.NewInt(int64(msg.Nonce())))
		// a.vmContext.Time.Add(a.vmContext.Time, big.NewInt(int64(msg.Nonce()*10)))
		evm := vm.NewEVM(a.vmContext, txContext, stateDB, a.chainConfig, vm.Config{
			Debug:  true,
			Tracer: tracer,
		})

		stateDB.(*apestate.OverlayStateDB).StartLogCollection(tx.Hash(), blockHash)
		msgResult, err := core.ApplyMessage(evm, msg, a.gasPool)
		// spew.Dump(msgResult)
		if err != nil {
			golog.Errorf("rejected tx: %s, from: %s, err: %v", tx.Hash().Hex(), msg.From(), err)
			stateDB.(*apestate.OverlayStateDB).RevertToSnapshot(snapshot)
			continue
		}
		if len(msgResult.Revert()) > 0 || msgResult.Err != nil {
			spew.Dump(msgResult.Revert(), msgResult.Err)
			reason, errUnpack := abi.UnpackRevert(msgResult.Revert())
			err = errors.New("execution reverted")
			if errUnpack == nil {
				err = fmt.Errorf("execution reverted: %v", reason)
			}
			execResults[i] = err
			golog.Errorf("TxIdx: %d,  err: %v", txIndex, err)
		}
		gasUsed += msgResult.UsedGas

		receipt := &types.Receipt{Type: tx.Type(), PostState: rootHash.Bytes(), CumulativeGasUsed: gasUsed}
		if msgResult.Failed() {
			receipt.Status = types.ReceiptStatusFailed
		} else {
			receipt.Status = types.ReceiptStatusSuccessful
		}
		receipt.TxHash = tx.Hash()
		receipt.BlockHash = blockHash
		receipt.BlockNumber = a.vmContext.BlockNumber
		receipt.GasUsed = msgResult.UsedGas

		if msg.To() == nil {
			receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
		}

		traceResult, err := tracer.(*tracers.Tracer).GetResult()
		if err != nil {
			golog.Error(err)
		}

		txExecutionLogs := stateDB.(*apestate.OverlayStateDB).GetLogs(tx.Hash())
		traceLogs := &types.Log{
			Address: common.HexToAddress("0xa9e5afe700000000a9e5afe700000000a9e5afe7"),
			Topics:  []common.Hash{crypto.Keccak256Hash([]byte("TRACE"))},
			Data:    traceResult,
		}
		receipt.Logs = append(txExecutionLogs, traceLogs)
		receipt.TransactionIndex = uint(txIndex)
		// spew.Dump(receipt)
		stateDB.(*apestate.OverlayStateDB).AddLog(traceLogs)
		stateDB.(*apestate.OverlayStateDB).FinishLogCollection()

		stateDB.(*apestate.OverlayStateDB).AddReceipt(tx.Hash(), receipt)
		// log.Printf("exec final depth: %d, snapshot revision id: %d", stateDB.(*apestate.OverlayStateDB).GetOverlayDepth(), snapshot)
		// stateDB.(*apestate.OverlayStateDB).MergeTo(1)
		txIndex++

		// writer.Write(traceResult)
		// writer.Flush()
	}

	return
}

func (a *ApeEVM) DoCall(msg *types.Message, debug bool, stateDB vm.StateDB) (*core.ExecutionResult, error) {
	txContext := core.NewEVMTxContext(msg)

	// a.callMutex.Lock()
	// log.Printf("DoCall clone from depth: %d", a.StateDB.GetOverlayDepth())
	// clonedDB := a.StateDB.Clone()

	vmCfg := vm.Config{
		Debug:  debug,
		Tracer: a.tracer,
	}

	stateDB.(*apestate.OverlayStateDB).SetCodeHash(msg.From(), common.Hash{})
	evm := vm.NewEVM(a.vmContext, txContext, stateDB, a.chainConfig, vmCfg)

	gasPool := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gasPool)
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}

	return result, nil
}
