package apebackend

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/dynm/ape-safer/apetracer"
	"github.com/dynm/ape-safer/constant"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ProbeAPI struct {
	b *ApeBackend
}

func (p *ProbeAPI) RunTxWithDifferentContext(ctx context.Context, txHash common.Hash) (interface{}, error) {
	txs, _ := p.b.TxPool.GetPoolTxs()
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

	// Run the transaction with tracing enabled.

	stateDB := p.b.EVM.StateDB.CloneFromRoot()

	stateDB.SetCodeHash(p.b.ImpersonatedAccount, common.Hash{})
	stateDB.SetCode(p.b.ImpersonatedAccount, []byte{})
	stateDB.AddBalance(constant.FAKE_ACCOUNT_0, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	stateDB.AddBalance(constant.FAKE_ACCOUNT_1, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	stateDB.AddBalance(constant.FAKE_ACCOUNT_2, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))
	stateDB.AddBalance(constant.FAKE_ACCOUNT_3, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000)))

	p.b.EVM.ExecuteTxs(txs, stateDB)
	msg := p.b.EVM.TxToMessage(txToBeTraced)

	txContext := core.NewEVMTxContext(msg)

	tracer := apetracer.NewKeccakTracer()
	vmCfg := vm.Config{
		Debug:  true,
		Tracer: tracer,
	}

	vmCtx := p.b.EVM.GetVMContext()
	chainCfg := p.b.EVM.GetChainConfig()

	evm := vm.NewEVM(vmCtx, txContext, stateDB.Clone(), &chainCfg, vmCfg)
	gasPool := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gasPool)
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}
	ops0 := tracer.GetResult()

	tracer.Reset()
	vmCtx.Difficulty.Add(vmCtx.Difficulty, big.NewInt(1))
	vmCtx.Time.Add(vmCtx.Time, big.NewInt(1))
	vmCtx.GasLimit += 1
	evm = vm.NewEVM(vmCtx, txContext, stateDB.Clone(), &chainCfg, vmCfg)
	gasPool = new(core.GasPool).AddGas(math.MaxUint64)
	result, err = core.ApplyMessage(evm, msg, gasPool)
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}
	ops1 := tracer.GetResult()

	// if len(ops0) != len(ops1) {
	// 	log.Printf("trace len does not match, len[0] = %d, len[1] = %d", len(ops0), len(ops1))
	// }
	// for i := range ops0 {
	// 	if ops0[i].Hash != ops1[i].Hash {
	// 		log.Printf("[0] pre: 0x%02x, hash: %s", ops0[i].Preimage, ops0[i].Hash.Hex())
	// 		log.Printf("[1] pre: 0x%02x, hash: %s", ops1[i].Preimage, ops1[i].Hash.Hex())
	// 	}
	// }

	// for i := range sOps0 {
	// 	if sOps0[i].Key == sOps1[i].Key && sOps0[i].Value == sOps1[i].Value {
	// 		continue
	// 	}
	// 	rw := ""
	// 	if sOps0[i].IsWrite {
	// 		rw = "W"
	// 	} else {
	// 		rw = "R"
	// 	}
	// 	log.Printf("[0] [%s] key: %s, val: %s", rw, sOps0[i].Key.Hex(), sOps0[i].Value.Hex())
	// 	if sOps1[i].IsWrite {
	// 		rw = "W"
	// 	} else {
	// 		rw = "R"
	// 	}
	// 	log.Printf("[1] [%s] key: %s, val: %s", rw, sOps1[i].Key.Hex(), sOps1[i].Value.Hex())
	// }
	traces := make(map[string][]interface{})
	traces["ops0"] = ops0
	traces["ops1"] = ops1
	return traces, nil
	return result, nil
}
