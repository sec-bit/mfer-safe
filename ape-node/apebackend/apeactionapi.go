package apebackend

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/big"

	"github.com/davecgh/go-spew/spew"
	"github.com/dynm/ape-safer/apesigner"
	"github.com/dynm/ape-safer/multisend"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/kataras/golog"
)

type ApeActionAPI struct {
	b *ApeBackend
}

func (s *ApeActionAPI) resetState() {
	s.b.EVM.ResetState()
	s.b.EVM.Prepare(nil)
}

func (s *ApeActionAPI) ResetState() {
	s.b.EVM.StateLock()
	defer s.b.EVM.StateUnlock()
	s.resetState()
}

func (s *ApeActionAPI) ClearTxPool() {
	s.b.TxPool.Reset()
	s.b.EVM.StateLock()
	defer s.b.EVM.StateUnlock()
	s.resetState()
}

func (s *ApeActionAPI) ReExecTxPool() {
	s.b.EVM.StateLock()
	defer s.b.EVM.StateUnlock()
	s.resetState()
	txs, _ := s.b.TxPool.GetPoolTxs()
	execResults := s.b.EVM.ExecuteTxs(txs, s.b.EVM.StateDB, nil)
	s.b.TxPool.SetResults(execResults)
}

func (s *ApeActionAPI) SetTimeDelta(delta uint64) {
	golog.Infof("Setting time delta to %d", delta)
	s.b.EVM.SetTimeDelta(delta)
}

func (s *ApeActionAPI) Impersonate(account common.Address) {
	s.b.ImpersonatedAccount = account
}

func (s *ApeActionAPI) SetBatchSize(batchSize int) {
	golog.Infof("Setting batch size to %d", batchSize)
	s.b.EVM.StateDB.SetBatchSize(batchSize)
}

func (s *ApeActionAPI) SetBlockNumberDelta(delta uint64) {
	golog.Infof("Setting block number delta to %d", delta)
	s.b.EVM.SetBlockNumberDelta(delta)
}

func (s *ApeActionAPI) PrintMoney(account common.Address) {
	s.b.EVM.StateDB.AddBalance(account, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
}

type TxData struct {
	Idx          int            `json:"idx"`
	From         common.Address `json:"from"`
	To           common.Address `json:"to"`
	Data         hexutil.Bytes  `json:"calldata"`
	ExecResult   string         `json:"execResult"`
	PseudoTxHash common.Hash    `json:"pseudoTxHash"`
}

type MultiSendData struct {
	TxData              []*TxData             `json:"txs"`
	MultiSendCallData   hexutil.Bytes         `json:"multisendCalldata"`
	MultiSendTxDataHash common.Hash           `json:"dataHash"`
	ApproveHashCallData hexutil.Bytes         `json:"approveHashCallData"`
	To                  common.Address        `json:"to"`
	SafeNonce           int64                 `json:"safeNonce"`
	ExecResult          *core.ExecutionResult `json:"execResult"`
	RevertError         string                `json:"revertError"`
	CallError           error                 `json:"callError"`
	EventLogs           []*types.Log          `json:"eventLogs"`
	DebugTrace          json.RawMessage       `json:"debugTrace"`
}

func NewApeActionAPI(b *ApeBackend) *ApeActionAPI {
	api := &ApeActionAPI{b}
	return api
}

func (s *ApeActionAPI) GetTxs() ([]*TxData, error) {
	txs, execResult := s.b.TxPool.GetPoolTxs()
	txData := make([]*TxData, len(txs))
	for i, tx := range txs {
		var to common.Address
		if tx.To() == nil {
			to = common.Address{}
		} else {
			to = *tx.To()
		}

		var result string
		if execResult[i] != nil {
			result = execResult[i].Error()
		}

		msg := s.b.EVM.TxToMessage(tx)
		txData[i] = &TxData{
			Idx:          i,
			From:         msg.From(),
			To:           to,
			Data:         tx.Data(),
			ExecResult:   result,
			PseudoTxHash: tx.Hash(),
		}
	}

	return txData, nil
}

func (s *ApeActionAPI) getSafeOwnersAndThreshold(safeAddr common.Address) ([]common.Address, int, error) {
	safe, err := multisend.NewGnosisSafe(safeAddr, s.b.EVM.SelfConn)
	if err != nil {
		return nil, 0, err
	}
	threshold, err := safe.GetThreshold(nil)
	if err != nil {
		return nil, 0, err
	}
	owners, err := safe.GetOwners(nil)
	if err != nil {
		return nil, 0, err
	}
	return owners, int(threshold.Int64()), nil
}

type SafeOwnerInfo struct {
	Owners    []common.Address `json:"owners"`
	Threshold int              `json:"threshold"`
}

func (s *ApeActionAPI) GetSafeOwnersAndThreshold() (*SafeOwnerInfo, error) {
	owners, threshold, err := s.getSafeOwnersAndThreshold(s.b.ImpersonatedAccount)
	if err != nil {
		return nil, err
	}
	return &SafeOwnerInfo{Owners: owners, Threshold: threshold}, nil
}

func (s *ApeActionAPI) SimulateSafeExec(ctx context.Context, safeOwners []common.Address) (*MultiSendData, error) {
	safeAddr := s.b.ImpersonatedAccount
	txs, _ := s.b.TxPool.GetPoolTxs()
	txData := make([]*TxData, len(txs))
	for i, tx := range txs {
		var to common.Address
		if tx.To() == nil {
			to = common.Address{}
		} else {
			to = *tx.To()
		}
		txData[i] = &TxData{
			Idx:  i,
			To:   to,
			Data: tx.Data(),
		}
	}

	calldata := multisend.BuildTransactions(txs)
	ms, err := multisend.NewMultisendSafe(s.b.EVM.Conn, safeAddr, multisend.MultiSendCallOnlyContractAddress, calldata, big.NewInt(0))
	if err != nil {
		return nil, err
	}

	nonce, err := ms.GetNonce()
	if err != nil {
		return nil, err
	}
	txDataHash, err := ms.GetTxDataHash(nonce.Int64())
	if err != nil {
		return nil, err
	}

	if len(safeOwners) == 0 {
		owners, threshold, err := s.getSafeOwnersAndThreshold(s.b.ImpersonatedAccount)
		if err != nil {
			return nil, err
		}
		safeOwners = owners[:threshold]
	}
	safeTx, err := ms.GenSafeCalldataWithApproveHash(safeOwners)
	if err != nil {
		return nil, err
	}
	for i, safeOwner := range safeOwners {
		golog.Infof("safeOwner[%d]: %s", i, safeOwner.Hex())
	}

	// s.b.EVM.StateDB.InitState()
	simulationStateDB := s.b.EVM.StateDB.CloneFromRoot()

	msData := &MultiSendData{
		TxData:              txData,
		MultiSendCallData:   hexutil.Bytes(safeTx),
		MultiSendTxDataHash: txDataHash,
		ApproveHashCallData: append([]byte{0xd4, 0xd9, 0xbd, 0xcd}, txDataHash.Bytes()...),
		To:                  safeAddr,
		SafeNonce:           nonce.Int64(),
	}

	signer := apesigner.NewSigner(s.b.EVM.ChainID().Int64())

	// approveHash
	safeOwnersNonce := make([]uint64, len(safeOwners))
	for i, safeOwner := range safeOwners {
		nonce, err := s.b.EVM.Conn.NonceAt(context.Background(), safeOwner, nil)
		if err != nil {
			return nil, err
		}
		safeOwnersNonce[i] = nonce
		simulationStateDB.AddBalance(safeOwner, big.NewInt(1e18))
		calldata := append(common.Hex2Bytes("d4d9bdcd"), msData.MultiSendTxDataHash.Bytes()...)
		tx := types.NewTransaction(nonce, s.b.ImpersonatedAccount, nil, 100_000, big.NewInt(5e9), calldata)
		tx, err = tx.WithSignature(signer, safeOwner.Bytes())
		if err != nil {
			log.Panic(err)
		}
		s.b.EVM.ExecuteTxs(types.Transactions{tx}, simulationStateDB, nil)
	}
	msg := types.NewMessage(
		safeOwners[0],
		&(s.b.ImpersonatedAccount),
		safeOwnersNonce[0],
		big.NewInt(0),
		5e6,
		big.NewInt(5e9),
		big.NewInt(0),
		big.NewInt(0),
		msData.MultiSendCallData,
		nil,
		true,
	)

	tracer, err := tracers.New("callTracer", new(tracers.Context))
	if err != nil {
		log.Panic(err)
	}

	s.b.EVM.SetTracer(tracer)
	txHash := crypto.Keccak256Hash([]byte("psuedoTransaction"))
	simulationStateDB.StartLogCollection(txHash, crypto.Keccak256Hash([]byte("blockhash")))
	result, err := s.b.EVM.DoCall(&msg, true, simulationStateDB)
	simulationStateDB.FinishLogCollection()
	spew.Dump(result, err)
	msData.ExecResult = result
	if err != nil {
		msData.CallError = err
	}

	if len(result.Revert()) > 0 {
		msData.RevertError = newRevertError(result).error.Error()
	}

	traceResult, err := tracer.GetResult()
	if err != nil {
		return nil, err
	}
	msData.DebugTrace = traceResult
	msData.EventLogs = simulationStateDB.GetLogs(txHash)

	return msData, nil

}

type txTraceResult struct {
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

func (s *ApeActionAPI) traceBlocks(ctx context.Context, blocks []*types.Block, config *tracers.TraceConfig) ([][]*txTraceResult, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no blocks supplied")
	}
	txTraceResults := make([][]*txTraceResult, len(blocks))

	spew.Dump(config)

	allTxs := make([]*types.Transaction, 0)
	for _, blk := range blocks {
		allTxs = append(allTxs, blk.Transactions()...)
	}

	// Assemble the structured logger or the JavaScript tracer

	stateBN := int(blocks[0].Header().Number.Int64() - 1)
	s.b.EVM.Prepare(&stateBN)
	BNu64 := uint64(stateBN)
	s.b.EVM.StateDB.InitState(&BNu64)
	stateDB := s.b.EVM.StateDB.CloneFromRoot()

	golog.Infof("Warming up %d txs", len(allTxs))
	s.b.EVM.WarmUpCache(allTxs, stateDB)
	golog.Info("Warmed up")

	stateDB = s.b.EVM.StateDB.CloneFromRoot()
	golog.Infof("Tracing: block from %d using state %d\n", blocks[0].Header().Number, stateBN)
	for i, block := range blocks {
		txs := block.Transactions()
		s.b.EVM.SetVMContextByBlock(block)
		s.b.EVM.ExecuteTxs(txs, stateDB, config)

		results := make([]*txTraceResult, len(txs))
		for i, tx := range txs {
			receipt := stateDB.GetReceipt(tx.Hash())
			if len(receipt.Logs) > 0 {
				trace := receipt.Logs[len(receipt.Logs)-1].Data
				results[i] = &txTraceResult{
					Result: json.RawMessage(trace),
				}
			}
		}
		txTraceResults[i] = results
	}

	// Run the transaction with tracing enabled.

	return txTraceResults, nil
}

func (s *ApeActionAPI) TraceBlockByNumber(ctx context.Context, number rpc.BlockNumber, config *tracers.TraceConfig) ([]*txTraceResult, error) {
	golog.Infof("tracing block number: %d", number)
	var bn *big.Int
	if number != -1 {
		bn = big.NewInt(number.Int64())
	}
	blk, err := s.b.EVM.Conn.BlockByNumber(ctx, bn)
	if err != nil {
		return nil, err
	}
	results, err := s.traceBlocks(ctx, []*types.Block{blk}, config)
	return results[0], err
}

func (s *ApeActionAPI) TraceBlockByNumberRange(ctx context.Context, numberFrom, numberTo rpc.BlockNumber, config *tracers.TraceConfig) ([][]*txTraceResult, error) {
	golog.Infof("tracing block number range: %d-%d", numberFrom, numberTo)
	var bnFrom, bnTo *big.Int
	if numberFrom != -1 {
		bnFrom = big.NewInt(numberFrom.Int64())
	}
	if numberTo != -1 {
		bnTo = big.NewInt(numberTo.Int64())
	}
	blockCnt := bnTo.Int64() - bnFrom.Int64() + 1
	blks := make([]*types.Block, blockCnt)
	for i := int64(0); i < blockCnt; i++ {
		golog.Infof("Fetching block %d", i)
		blk, err := s.b.EVM.Conn.BlockByNumber(ctx, big.NewInt(i+bnFrom.Int64()))
		if err != nil {
			return nil, err
		}
		blks[i] = blk
	}
	results, err := s.traceBlocks(ctx, blks, config)
	// spew.Dump(results)
	return results, err
}
