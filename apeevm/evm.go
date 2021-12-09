package apeevm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/eth/tracers"

	"github.com/dustin/go-humanize"
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
	ctx       context.Context
	RpcClient *rpc.Client
	Conn      *ethclient.Client
	SelfConn  *ethclient.Client

	StateDB             *apestate.OverlayStateDB
	signer              types.Signer
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

func NewApeEVM(rawurl string, impersonatedAccount common.Address) *ApeEVM {
	apeEVM := &ApeEVM{}
	ctx := context.Background()
	RpcClient, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		log.Panic(err)
	}
	apeEVM.ctx = ctx
	apeEVM.RpcClient = RpcClient
	apeEVM.Conn = ethclient.NewClient(RpcClient)
	apeEVM.callMutex = &sync.RWMutex{}
	apeEVM.stateLock = &sync.RWMutex{}
	apeEVM.impersonatedAccount = impersonatedAccount
	apeEVM.Prepare()

	go apeEVM.updatePendingBN()

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
		log.Printf("GetBlockHeader err: %v", err)
		return nil
	} else if len(raw) == 0 {
		log.Printf("GetBlockHeader: Block not found")
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
	a.StateDB.CloseCache()
	a.gasPool = new(core.GasPool)
	a.gasPool.AddGas(lastBlockHeader.GasLimit)
}

func (a *ApeEVM) ChainID() *big.Int {
	return a.chainConfig.ChainID
}

func (a *ApeEVM) Prepare() {
	a.chainConfig = core.DefaultGenesisBlock().Config
	chainID, err := a.Conn.ChainID(a.ctx)
	if err != nil {
		log.Panic(err)
	}
	a.chainConfig.ChainID = chainID

	//avoid invalid opcode: SHR
	a.chainConfig.ByzantiumBlock = big.NewInt(0)
	a.chainConfig.ConstantinopleBlock = big.NewInt(0)

	lastBlockHeader := a.GetLatestBlockHeader()
	if lastBlockHeader == nil {
		log.Panic("[Prepare] cannot get last block")
	}

	a.StateDB = apestate.NewOverlayStateDB(a.RpcClient, int(lastBlockHeader.Number.Uint64()))
	// a.StateDB = apestate.NewApeStateDB(a.RpcClient, int(lastBlock.NumberU64()))
	a.StateDB.SetCodeHash(a.impersonatedAccount, common.Hash{})
	a.StateDB.SetCode(a.impersonatedAccount, []byte{})
	a.StateDB.AddBalance(constant.FAKE_ACCOUNT_0, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	a.StateDB.AddBalance(constant.FAKE_ACCOUNT_1, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	a.StateDB.AddBalance(constant.FAKE_ACCOUNT_2, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	a.StateDB.AddBalance(constant.FAKE_ACCOUNT_3, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))

	a.signer = apesigner.NewSigner(a.ChainID().Int64())

	getHash := func(bn uint64) common.Hash {
		blk, err := a.Conn.BlockByNumber(a.ctx, new(big.Int).SetUint64(bn))
		if err != nil {
			return common.Hash{}
		}
		return blk.Hash()
	}
	a.gasPool = new(core.GasPool)
	a.gasPool.AddGas(lastBlockHeader.GasLimit)
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
	a.setVMContext()
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

func (a *ApeEVM) GetVMContext() vm.BlockContext {
	return a.vmContext
}

func (a *ApeEVM) updatePendingBN() {
	ticker5Sec := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker5Sec.C:
			a.setVMContext()
			sizeStr := humanize.Bytes(uint64(a.StateDB.CacheSize()))
			log.Printf("[Update] BN: %d, Ts: %d, Diff: %d, GasLimit: %d, Cache: %s, RPCReq: %d, StateBlock: %d",
				a.vmContext.BlockNumber, a.vmContext.Time, a.vmContext.Difficulty, a.vmContext.GasLimit, sizeStr, a.StateDB.RPCRequestCount(), a.StateDB.StateBlockNumber())
		}
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
		signer = a.signer
	} else {
		signer = types.NewLondonSigner(a.ChainID())
	}
	msg, _ := tx.AsMessage(signer, big.NewInt(10e9))
	return msg
}

func (a *ApeEVM) ExecuteTxs(txs types.Transactions, stateDB vm.StateDB) (execResults []error) {
	execResults = make([]error, len(txs))
	var (
		gasUsed = uint64(0)
		txIndex = 0
	)

	for i, tx := range txs {
		// spew.Dump(tx)
		msg := a.TxToMessage(tx)
		log.Printf("From: %s, To: %s, Nonce: %d, GasPrice: %d, Gas: %d, Hash: %s", msg.From(), msg.To(), msg.Nonce(), msg.GasPrice(), msg.Gas(), tx.Hash())

		txContext := core.NewEVMTxContext(msg)
		snapshot := stateDB.Snapshot()
		tracer, err := tracers.New("callTracer", new(tracers.Context))
		if err != nil {
			log.Panic(err)
		}

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
			log.Printf("rejected tx: %s, from: %s, err: %v", tx.Hash().Hex(), msg.From(), err)
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
			log.Printf("TxIdx: %d,  err: %v", txIndex, err)
		}
		stateDB.(*apestate.OverlayStateDB).FinishLogCollection()
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

		traceResult, err := tracer.GetResult()
		if err != nil {
			log.Print(err)
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
		stateDB.(*apestate.OverlayStateDB).AddReceipt(tx.Hash(), receipt)
		log.Printf("exec final depth: %d, snapshot revision id: %d", stateDB.(*apestate.OverlayStateDB).GetOverlayDepth(), snapshot)
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

	evm := vm.NewEVM(a.vmContext, txContext, stateDB, a.chainConfig, vmCfg)

	gasPool := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gasPool)
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}

	return result, nil
}
