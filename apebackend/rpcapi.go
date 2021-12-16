package apebackend

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/davecgh/go-spew/spew"
	"github.com/dynm/ape-safer/apesigner"
	"github.com/dynm/ape-safer/apetracer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

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
		{
			Namespace: "probe",
			Version:   "1.0",
			Service:   &ProbeAPI{b},
			Public:    true,
		},
	}
}

type ApeAPI struct {
	b *ApeBackend
}

type AuxAPI struct {
	b *ApeBackend
}

func (s *AuxAPI) Version() string {
	return s.b.EVM.ChainID().String()
}

func (s *AuxAPI) Listening() bool {
	return true
}

type ChainIDArgs struct {
	ChainID *hexutil.Big `json:"balance"`
}

func (s *AuxAPI) SwitchEthereumChain(args ChainIDArgs) {
	// s.b.EVM.ChainID()
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
		msg := s.b.EVM.TxToMessage(tx)
		rpcTx.From = msg.From()
		return rpcTx, nil
	}

	// Transaction unknown, return as such
	return nil, nil
}

func (s *ApeAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := s.b.EVM.Conn.BlockByHash(ctx, hash)
	if block != nil {
		return RPCMarshalBlock(block, true, fullTx)
	} else {
		block, err = s.b.EVM.Conn.BlockByNumber(ctx, nil)
		if err != nil {
			return nil, err
		}
		response, err := RPCMarshalBlock(block, true, false)
		if err != nil {
			return nil, err
		}
		return response, nil
	}
	return nil, err
}

func (s *ApeAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	switch number {
	case rpc.LatestBlockNumber:
		{
			prevBlock, err := s.b.EVM.Conn.BlockByNumber(ctx, nil)
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
			response["hash"] = common.HexToHash("0xcafecafecafecafecafecafecafecafecafecafecafecafecafecafecafecafe")

		}
	case rpc.PendingBlockNumber:
		return response, nil
	default:
		block, err := s.b.EVM.Conn.BlockByNumber(ctx, big.NewInt(int64(number)))
		if err != nil {
			return nil, err
		}
		response, err = RPCMarshalBlock(block, true, false)
		if err == nil && number == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce"} {
				response[field] = nil
			}
		}
	}

	response["miner"] = common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	response["totalDifficulty"] = "0xcafebabe0a9e5afe"
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
