package mferbackend

import (
	"github.com/dynm/mfer-safe/constant"
	"github.com/dynm/mfer-safe/mferevm"
	"github.com/dynm/mfer-safe/mfertxpool"
	"github.com/ethereum/go-ethereum/common"
)

type MferBackend struct {
	EVM                 *mferevm.MferEVM
	TxPool              *mfertxpool.MferTxPool
	ImpersonatedAccount common.Address
}

func NewMferBackend(e *mferevm.MferEVM, txPool *mfertxpool.MferTxPool, impersonatedAccount common.Address) *MferBackend {
	return &MferBackend{
		EVM:                 e,
		TxPool:              txPool,
		ImpersonatedAccount: impersonatedAccount,
	}
}

func (b *MferBackend) Accounts() []common.Address {
	return []common.Address{
		b.ImpersonatedAccount,
		constant.FAKE_ACCOUNT_0,
		constant.FAKE_ACCOUNT_1,
		constant.FAKE_ACCOUNT_2,
		constant.FAKE_ACCOUNT_3,
	}
}
