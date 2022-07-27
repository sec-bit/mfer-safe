package mferevm

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestEVMExecute(t *testing.T) {
	mferEVM := NewMferEVM("http://tractor.local:8545", common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "./keycache.txt", 50)
	mferEVM.Prepare(nil)

	tx, _, _ := mferEVM.Conn.TransactionByHash(context.Background(), common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	txs := make(types.Transactions, 2)
	txs[0] = tx
	txs[1] = tx

	mferEVM.ExecuteTxs(txs, nil, nil)
}
