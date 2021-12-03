package apebackend

import (
	"context"
	"encoding/json"
	"log"
	"math/big"

	"github.com/davecgh/go-spew/spew"
	"github.com/dynm/ape-safer/apesigner"
	"github.com/dynm/ape-safer/apestate"
	"github.com/dynm/ape-safer/multisend"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

type ApeActionAPI struct {
	b *ApeBackend
}

func (s *ApeActionAPI) resetState() {
	s.b.EVM.ResetState()
	s.b.EVM.Prepare()
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
	execResults := s.b.EVM.ExecuteTxs(txs, s.b.EVM.StateDB)
	s.b.TxPool.SetResults(execResults)
}

func (s *ApeActionAPI) SetTimeDelta(delta uint64) {
	s.b.EVM.SetTimeDelta(delta)
}

func (s *ApeActionAPI) SetBlockNumberDelta(delta uint64) {
	s.b.EVM.SetBlockNumberDelta(delta)
}

type TxData struct {
	Idx        int            `json:"idx"`
	To         common.Address `json:"to"`
	Data       hexutil.Bytes  `json:"calldata"`
	ExecResult string         `json:"execResult"`
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
		txData[i] = &TxData{
			Idx:        i,
			To:         to,
			Data:       tx.Data(),
			ExecResult: result,
		}
	}

	return txData, nil
}

func (s *ApeActionAPI) getSafeOwnersAndThreshold(safeAddr common.Address) ([]common.Address, int, error) {
	safe, err := multisend.NewGnosisSafe(safeAddr, s.b.EVM.Conn)
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
		log.Printf("safeOwner[%d]: %s", i, safeOwner.Hex())
	}

	latestHeader := s.b.EVM.GetLatestBlockHeader()
	simulationStateDB := apestate.NewOverlayStateDB(s.b.EVM.RpcClient, int(latestHeader.Number.Int64()))

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
		s.b.EVM.ExecuteTxs(types.Transactions{tx}, simulationStateDB)
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
	result, err := s.b.EVM.DoCall(&msg, true, simulationStateDB)
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

	return msData, nil

}
