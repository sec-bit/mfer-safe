package mfertxpool

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type MferTxPool struct {
	txs         types.Transactions
	execResults []error
}

func NewMferTxPool() *MferTxPool {
	pool := &MferTxPool{
		txs:         make(types.Transactions, 0),
		execResults: make([]error, 0),
	}
	return pool
}

func (pool *MferTxPool) AddTx(tx *types.Transaction, execResult error) {
	pool.txs = append(pool.txs, tx)
	pool.execResults = append(pool.execResults, execResult)
}

func (pool *MferTxPool) SetResults(execResults []error) {
	pool.execResults = execResults
}

func (pool *MferTxPool) Reset() (n int) {
	n = len(pool.txs)
	pool.txs = make(types.Transactions, 0)
	pool.execResults = make([]error, 0)
	return
}

func (pool *MferTxPool) RemoveTxByHash(txHash common.Hash) {
	if len(pool.txs) < 1 {
		return
	}
	txIndex := 0
	for i, tx := range pool.txs {
		if tx.Hash() == txHash {
			txIndex = i
			break
		}
	}
	head := pool.txs[:txIndex]
	tail := pool.txs[txIndex+1:]
	pool.txs = append(head, tail...)

	resHead := pool.execResults[:txIndex]
	resTail := pool.execResults[txIndex+1:]
	pool.execResults = append(resHead, resTail...)
}

func (pool *MferTxPool) GetPoolTxs() (types.Transactions, []error) {
	return pool.txs, pool.execResults
}

func (pool *MferTxPool) GetTransactionByHash(txHash common.Hash) (int, *types.Transaction) {
	for i, tx := range pool.txs {
		if tx.Hash() == txHash {
			return i, tx
		}
	}
	return 0, nil
}
