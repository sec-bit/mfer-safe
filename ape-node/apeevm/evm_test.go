package apeevm

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestEVMExecute(t *testing.T) {
	apeEVM := NewApeEVM("http://tractor.local:8545", common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	apeEVM.Prepare()

	tx, _, _ := apeEVM.Conn.TransactionByHash(context.Background(), common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	txs := make(types.Transactions, 2)
	txs[0] = tx
	txs[1] = tx

	apeEVM.ExecuteTxs(txs, nil)
}
