package apestate

import "github.com/ethereum/go-ethereum/common"

type SlotKey [20 + 32]byte

func (k *SlotKey) Account() common.Address {
	return common.BytesToAddress(k[:20])
}

func (k *SlotKey) Key() common.Hash {
	return common.BytesToHash(k[20 : 20+32])
}

func (k *SlotKey) Extract() (common.Address, common.Hash) {
	return k.Account(), k.Key()
}

func calcSlotKey(acc common.Address, key common.Hash) SlotKey {
	var slotKey SlotKey
	copy(slotKey[:20], acc.Bytes())
	copy(slotKey[20:20+32], key.Bytes())
	return slotKey
}
