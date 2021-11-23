package apebackend

import (
	"github.com/dynm/ape-safer/apeevm"
	"github.com/dynm/ape-safer/apetxpool"
	"github.com/dynm/ape-safer/constant"
	"github.com/ethereum/go-ethereum/common"
)

type ApeBackend struct {
	EVM                 *apeevm.ApeEVM
	TxPool              *apetxpool.ApeTxPool
	ImpersonatedAccount common.Address
}

func NewApeBackend(e *apeevm.ApeEVM, txPool *apetxpool.ApeTxPool, impersonatedAccount common.Address) *ApeBackend {
	return &ApeBackend{
		EVM:                 e,
		TxPool:              txPool,
		ImpersonatedAccount: impersonatedAccount,
	}
}

func (b *ApeBackend) Accounts() []common.Address {
	return []common.Address{
		b.ImpersonatedAccount,
		constant.FAKE_ACCOUNT_0,
		constant.FAKE_ACCOUNT_1,
		constant.FAKE_ACCOUNT_2,
		constant.FAKE_ACCOUNT_3,
	}
}
