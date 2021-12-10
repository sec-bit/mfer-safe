package apebackend

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

type ExecutionResult struct {
	Gas         uint64         `json:"gas"`
	Failed      bool           `json:"failed"`
	ReturnValue string         `json:"returnValue"`
	StructLogs  []StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc      uint64             `json:"pc"`
	Op      string             `json:"op"`
	Gas     uint64             `json:"gas"`
	GasCost uint64             `json:"gasCost"`
	Depth   int                `json:"depth"`
	Error   string             `json:"error,omitempty"`
	Stack   *[]string          `json:"stack,omitempty"`
	Memory  *[]string          `json:"memory,omitempty"`
	Storage *map[string]string `json:"storage,omitempty"`
}

// FormatLogs formats EVM returned structured logs for json output
func FormatLogs(logs []vm.StructLog) []StructLogRes {
	formatted := make([]StructLogRes, len(logs))
	for index, trace := range logs {
		formatted[index] = StructLogRes{
			Pc:      trace.Pc,
			Op:      trace.Op.String(),
			Gas:     trace.Gas,
			GasCost: trace.GasCost,
			Depth:   trace.Depth,
			Error:   trace.ErrorString(),
		}
		if trace.Stack != nil {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = stackValue.Hex()
			}
			formatted[index].Stack = &stack
		}
		if trace.Memory != nil {
			memory := make([]string, 0, (len(trace.Memory)+31)/32)
			for i := 0; i+32 <= len(trace.Memory); i += 32 {
				memory = append(memory, fmt.Sprintf("%x", trace.Memory[i:i+32]))
			}
			formatted[index].Memory = &memory
		}
		if trace.Storage != nil {
			storage := make(map[string]string)
			for i, storageValue := range trace.Storage {
				storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
			}
			formatted[index].Storage = &storage
		}
	}
	return formatted
}

type DebugAPI struct {
	b *ApeBackend
}

func (s *DebugAPI) TraceTransaction(ctx context.Context, txHash common.Hash, config *tracers.TraceConfig) (interface{}, error) {
	spew.Dump(config)
	txs, _ := s.b.TxPool.GetPoolTxs()
	// retrive all previous txs for state mutation
	var txToBeTraced *types.Transaction
	for i, tx := range txs {
		if tx.Hash() == txHash {
			txToBeTraced = tx
			txs = txs[:i]
			log.Printf("found: tx[%d], head len: %d", i, len(txs))
			break
		}
	}
	if txToBeTraced == nil {
		return nil, fmt.Errorf("tx %s not found", txHash.Hex())
	}

	// Assemble the structured logger or the JavaScript tracer
	var (
		tracer vm.Tracer
		err    error
	)

	txctx := &tracers.Context{
		BlockHash: s.b.EVM.GetLatestBlockHeader().ParentHash,
		TxIndex:   len(txs),
		TxHash:    txToBeTraced.Hash(),
	}

	switch {
	case config != nil && config.Tracer != nil:
		// Define a meaningful timeout of a single transaction trace
		timeout := time.Second * 1
		if config.Timeout != nil {
			if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
				return nil, err
			}
		}
		// Constuct the JavaScript tracer to execute with
		if tracer, err = tracers.New(*config.Tracer, txctx); err != nil {
			return nil, err
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
		tracer = vm.NewStructLogger(nil)
	default:
		config.LogConfig.EnableMemory = true
		config.LogConfig.EnableReturnData = true
		config.LogConfig.DisableStorage = false
		tracer = vm.NewStructLogger(config.LogConfig)
	}
	// Run the transaction with tracing enabled.

	stateDB := s.b.EVM.StateDB.CloneFromRoot()
	stateDB.InitFakeAccounts()
	s.b.EVM.ExecuteTxs(txs, stateDB)

	s.b.EVM.SetTracer(tracer)
	msg := s.b.EVM.TxToMessage(txToBeTraced)

	result, err := s.b.EVM.DoCall(&msg, true, stateDB)
	if err != nil {
		return nil, err
	}
	// Depending on the tracer type, format and return the output.
	switch tracer := tracer.(type) {
	case *vm.StructLogger:
		// If the result contains a revert reason, return it.
		returnVal := fmt.Sprintf("%x", result.Return())
		if len(result.Revert()) > 0 {
			returnVal = fmt.Sprintf("%x", result.Revert())
		}
		return &ExecutionResult{
			Gas:         result.UsedGas,
			Failed:      result.Failed(),
			ReturnValue: returnVal,
			StructLogs:  FormatLogs(tracer.StructLogs()),
		}, nil

	case *tracers.Tracer:
		return tracer.GetResult()

	default:
		panic(fmt.Sprintf("bad tracer type %T", tracer))
	}
}

func (s *DebugAPI) Preimage(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	// if preimage := s.b.EVM.StateDB.ReadPreimage(api.eth.ChainDb(), hash); preimage != nil {
	// 	return preimage, nil
	// }
	return nil, errors.New("unknown preimage")
}

type StorageRangeResult struct {
	Storage storageMap   `json:"storage"`
	NextKey *common.Hash `json:"nextKey"` // nil if Storage includes the last key in the trie.
}

type storageMap map[common.Hash]storageEntry

type storageEntry struct {
	Key   *common.Hash `json:"key"`
	Value common.Hash  `json:"value"`
}

// StorageRangeAt returns the storage at the given block height and transaction index.
func (s *DebugAPI) StorageRangeAt(blockHash common.Hash, txIndex int, contractAddress common.Address, keyStart hexutil.Bytes, maxResult int) (StorageRangeResult, error) {
	// Retrieve the block

	// if s.b.EVM.StateDB == nil {
	return StorageRangeResult{}, fmt.Errorf("account %x doesn't exist", contractAddress)
	// }
	// return storageRangeAt(s.b.EVM.StateDB, keyStart, maxResult)
}

// func storageRangeAt(st *apestate.OverlayStateDB, start []byte, maxResult int) (StorageRangeResult, error) {
// 	it := trie.NewIterator(st.NodeIterator(start))
// 	result := StorageRangeResult{Storage: storageMap{}}
// 	for i := 0; i < maxResult && it.Next(); i++ {
// 		_, content, _, err := rlp.Split(it.Value)
// 		if err != nil {
// 			return StorageRangeResult{}, err
// 		}
// 		e := storageEntry{Value: common.BytesToHash(content)}
// 		if preimage := st.GetKey(it.Key); preimage != nil {
// 			preimage := common.BytesToHash(preimage)
// 			e.Key = &preimage
// 		}
// 		result.Storage[common.BytesToHash(it.Key)] = e
// 	}
// 	// Add the 'next key' so clients can continue downloading.
// 	if it.Next() {
// 		next := common.BytesToHash(it.Key)
// 		result.NextKey = &next
// 	}
// 	return result, nil
// }
