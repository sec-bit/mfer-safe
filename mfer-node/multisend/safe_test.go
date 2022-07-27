package multisend

import (
	"log"
	"math/big"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func TestGenerateSafeExecTx(t *testing.T) {
	GenerateSafeExecTx(common.Address{}, common.Address{}, big.NewInt(0), nil, 0, nil)
}

func TestDecodeSafeExecTx(t *testing.T) {
	safeABI, err := abi.JSON(strings.NewReader(GnosisSafeABI))
	if err != nil {
		log.Panic(err)
	}
	/*
		to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte
	*/
	tx := common.Hex2Bytes("0x6a761202000000000000000000000000f70314eb9c7fe7d88e6af5aa7f898b3a162dcd48000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007be5e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000004496294178000000000000000000000000d055d32e50c57b413f7c2a4a052faf6933ea79270000000000000000000000000000000000000000000000000517064278dc91a70000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000823bcdc04d96cf9e074d7ccd2274b9016aa6e410694f6cfd328f49ef7cdd269593138b86fcc872cd150d3d7d1304f37c997119ebabc81d7ac3a75740563154f3c91ba3c648ae746c85b20b4ef23332b6fa778e335e56beed69b615a2b510e8f504837aa9a1a36d56242d8ae729cff390d8cc864da930e0109c86d0742a8b47eb2e5b1c000000000000000000000000000000000000000000000000000000000000"[10:])

	decoded, err := safeABI.Methods["execTransaction"].Inputs.Unpack(tx)
	spew.Dump(decoded)

}