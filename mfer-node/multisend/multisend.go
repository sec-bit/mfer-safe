package multisend

import (
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	MultiSendCallOnlyContractAddress = common.HexToAddress("0x40A2aCCbd92BCA938b02010E17A5b8929b49130D")
	MultiSendContractAddress         = common.HexToAddress("0xA238CBeb142c10Ef7Ad8442C6D1f9E89e07e7761")
)

func BuildTransactions(txs types.Transactions) (multisendArgs []byte) {

	for _, tx := range txs {
		multisendArgs = append(multisendArgs, BuildTransaction(tx)...)
	}

	multisendABI, err := abi.JSON(strings.NewReader(MultiSendCallOnlyABI))
	if err != nil {
		log.Panic(err)
	}
	selector := common.Hex2Bytes("8d80ff0a")
	ret, err := multisendABI.Methods["multiSend"].Inputs.Pack(multisendArgs)
	if err != nil {
		log.Panic(err)
	}

	return append(selector, ret...)
}

func BuildTransaction(tx *types.Transaction) (ret []byte) {
	/*
		00 //operation call
		f000000000000000000000000000012345678908 //to
		0000000000000000000000000000000000000000000000000000000000000000 //value
		0000000000000000000000000000000000000000000000000000000000000044 //datalen
		ffffffff000000000000000000000000f0000000000000000000000000000123456789080000000000000000000000000000000000000000000000000111111111110000 //data
	*/
	ret = append(ret, 0x00)
	ret = append(ret, tx.To().Bytes()...)
	valueB := make([]byte, 32)
	tx.Value().FillBytes(valueB)
	ret = append(ret, valueB...)
	data := tx.Data()
	dataLenB := make([]byte, 32)
	dataLenBN := big.NewInt(int64(len(data)))
	dataLenBN.FillBytes(dataLenB)
	ret = append(ret, dataLenB...)
	ret = append(ret, data...)

	return ret
}
