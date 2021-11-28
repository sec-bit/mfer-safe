package apebackend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/davecgh/go-spew/spew"
	"github.com/dynm/ape-safer/apesigner"
	"github.com/dynm/ape-safer/apestate"
	"github.com/dynm/ape-safer/apetracer"
	"github.com/dynm/ape-safer/multisend"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"
)

type ApeAPI struct {
	b *ApeBackend
}

type AuxAPI struct {
	b *ApeBackend
}

func (s *AuxAPI) Version() string {
	return s.b.EVM.ChainID().String()
}

type ChainIDArgs struct {
	ChainID *hexutil.Big `json:"balance"`
}

func (s *AuxAPI) SwitchEthereumChain(args ChainIDArgs) {
	// s.b.EVM.ChainID()
}

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

func GetApeAPIs(b *ApeBackend) []rpc.API {
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   &ApeAPI{b},
			Public:    true,
		},
		{
			Namespace: "net",
			Version:   "1.0",
			Service:   &AuxAPI{b},
			Public:    true,
		},
		{
			Namespace: "wallet",
			Version:   "1.0",
			Service:   &AuxAPI{b},
			Public:    true,
		},
		{
			Namespace: "debug",
			Version:   "1.0",
			Service:   &DebugAPI{b},
			Public:    true,
		},
		{
			Namespace: "ape",
			Version:   "1.0",
			Service:   NewApeActionAPI(b),
			Public:    true,
		},
	}
}

func NewApeActionAPI(b *ApeBackend) *ApeActionAPI {
	api := &ApeActionAPI{b}
	// go func() {
	// 	ticker2Min := time.NewTicker(time.Minute * 2)
	// 	for {
	// 		select {
	// 		case <-ticker2Min.C:
	// 			log.Printf("trigger auto reset")
	// 			api.ReExecTxPool()
	// 		}
	// 	}
	// }()
	return api
}

func (s *ApeAPI) Accounts() []common.Address {
	return s.b.Accounts()
}

func (s *ApeAPI) RequestAccounts() []common.Address {
	return s.b.Accounts()
}

func (s *ApeAPI) Call(ctx context.Context, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride) (hexutil.Bytes, error) {
	msg, err := args.ToMessage(0, nil)
	if err != nil {
		return nil, err
	}

	result, err := s.b.EVM.DoCall(&msg, false, s.b.EVM.StateDB.Clone())
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return nil, newRevertError(result)
	}
	return result.Return(), result.Err
}

func (s *ApeAPI) EstimateGas(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	var from *common.Address
	if args.From != nil {
		from = args.From
	} else {
		from = new(common.Address)
	}
	args.GasPrice = nil
	nonce := s.b.EVM.StateDB.GetNonce(*from)
	huNonce := hexutil.Uint64(nonce)
	args.Nonce = &huNonce
	msg, err := args.ToMessage(0, nil)
	if err != nil {
		return 0, err
	}
	tracer := &apetracer.KeccakTracer{}

	// tracer, err := tracers.New("callTracer", new(tracers.Context))
	// if err != nil {
	// 	log.Panic(err)
	// }
	s.b.EVM.SetTracer(tracer)
	result, err := s.b.EVM.DoCall(&msg, true, s.b.EVM.StateDB.Clone())
	// ret, resError := tracer.GetResult()
	// if resError != nil {
	// 	return 0, err
	// }
	// log.Printf("trace: %s", ret)
	if err != nil {
		return 0, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return hexutil.Uint64(result.UsedGas * 2), newRevertError(result)
	}
	return hexutil.Uint64(result.UsedGas * 2), nil
}

func (s *ApeAPI) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	state := s.b.EVM.StateDB

	if state == nil {
		return nil, fmt.Errorf("ape state not found")
	}
	return (*hexutil.Big)(state.GetBalance(address)), nil
}

func (s *ApeAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state := s.b.EVM.StateDB
	if state == nil {
		return nil, fmt.Errorf("ape state not found")
	}
	return (hexutil.Bytes)(state.GetCode(address)), nil
}

func (s *ApeAPI) SendTransaction(ctx context.Context, args TransactionArgs) (common.Hash, error) {
	var from *common.Address
	if args.From != nil && (*args.From).String() != (common.Address{}).String() {
		from = args.From
	} else {
		addr := s.b.ImpersonatedAccount
		from = &addr
	}

	gp := hexutil.Big(*big.NewInt(0))
	args.GasPrice = &gp

	s.b.EVM.StateLock()
	defer s.b.EVM.StateUnlock()
	if args.Gas == nil {
		gas, err := s.EstimateGas(ctx, args, nil)
		if err != nil {
			return common.Hash{}, err
		}
		args.Gas = &gas
	}

	nonce := s.b.EVM.StateDB.GetNonce(*from)
	args.Nonce = (*hexutil.Uint64)(&nonce)

	signer := apesigner.NewSigner(s.b.EVM.ChainID().Int64())

	spew.Dump(args)

	tx, err := args.ToTransaction().WithSignature(signer, from.Bytes())
	if err != nil {
		log.Panic(err)
	}
	res := s.b.EVM.ExecuteTxs(types.Transactions{tx}, s.b.EVM.StateDB)
	s.b.TxPool.AddTx(tx, res[0])
	return tx.Hash(), nil
}

func (s *ApeAPI) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return common.Hash{}, err
	}

	res := s.b.EVM.ExecuteTxs(types.Transactions{tx}, s.b.EVM.StateDB)
	s.b.TxPool.AddTx(tx, res[0])
	return tx.Hash(), nil
}

var (
	blockHash = crypto.Keccak256Hash([]byte("fake block hash"))
)

func (s *ApeAPI) GetTransactionByHash(ctx context.Context, hash common.Hash) (*RPCTransaction, error) {
	// Try to return an already finalized transaction
	index, tx := s.b.TxPool.GetTransactionByHash(hash)
	if tx == nil {
		return nil, fmt.Errorf("tx: %s not found", hash.Hex())
	}
	if tx != nil {
		rpcTx := newRPCTransaction(tx, blockHash, uint64(s.BlockNumber()), uint64(index), nil)
		rpcTx.From = s.b.ImpersonatedAccount
		return rpcTx, nil
	}

	// Transaction unknown, return as such
	return nil, nil
}

func (s *ApeAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	var bn *big.Int
	switch number {
	case rpc.LatestBlockNumber:
		bn = nil
	case rpc.PendingBlockNumber:
		latestBH := s.b.EVM.GetLatestBlockHeader()
		bn = latestBH.Number
	default:
		bn = big.NewInt(int64(number))
	}
	block, err := s.b.EVM.Conn.BlockByNumber(ctx, bn)
	if err != nil {
		return nil, err
	}
	var response map[string]interface{}
	if block != nil && err == nil {
		response, err = RPCMarshalBlock(block, true, false)
		if err == nil && number == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce"} {
				response[field] = nil
			}
		}
	} else {
		prevBlock, err := s.b.EVM.Conn.BlockByNumber(ctx, big.NewInt(number.Int64()-1))
		if err != nil {
			return nil, err
		}
		poolTxs, _ := s.b.TxPool.GetPoolTxs()
		prevHeader := prevBlock.Header()
		prevHeader.Number.Add(prevHeader.Number, big.NewInt(1))
		prevHeader.Time += 10
		currBlock := types.NewBlockWithHeader(prevHeader).WithBody(poolTxs, nil)
		response, err = RPCMarshalBlock(currBlock, true, false)
		if err != nil {
			return nil, err
		}
	}

	response["miner"] = common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	return response, nil

	// _ = block
	// ret := make(map[string]interface{})
	// ret["hash"] = block.Hash().Hex()
	// ret["timestamp"] = hexutil.Uint64(block.Time())
	// ret["miner"] = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	// return ret, nil
}

func (s *ApeAPI) GasPrice(ctx context.Context) (*hexutil.Big, error) {
	tipcap := big.NewInt(5e9)
	return (*hexutil.Big)(tipcap), nil
}

func (s *ApeAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	nonce := s.b.EVM.StateDB.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), nil
}

func (s *ApeAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	index, tx := s.b.TxPool.GetTransactionByHash(hash)
	if tx == nil {
		return nil, fmt.Errorf("tx: %s not found", hash.Hex())
	}

	receipt := s.b.EVM.StateDB.GetReceipt(hash)
	if receipt == nil {
		return nil, fmt.Errorf("tx: %s receipt not found", hash.Hex())
	}

	// Derive the sender.
	// bigblock := new(big.Int).SetUint64(blockNumber)
	// signer := types.MakeSigner(s.b.EVM.chainConfig, bigblock)
	// from, _ := types.Sender(signer, tx)
	from := s.b.ImpersonatedAccount
	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       s.BlockNumber(),
		"transactionHash":   hash,
		"transactionIndex":  hexutil.Uint64(index),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
		"type":              hexutil.Uint(tx.Type()),
	}

	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	}
	fields["status"] = hexutil.Uint(receipt.Status)

	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields, nil
}

func (s *ApeAPI) ChainId() (*hexutil.Big, error) {
	return (*hexutil.Big)(s.b.EVM.ChainID()), nil
}

func (s *ApeAPI) BlockNumber() hexutil.Uint64 {
	bn, err := s.b.EVM.Conn.BlockNumber(context.TODO())
	if err != nil {
		return hexutil.Uint64(0)
	}
	return hexutil.Uint64(bn + 1)
}
