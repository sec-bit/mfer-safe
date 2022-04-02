package multisend

import (
	"log"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Operation uint8

const (
	Call Operation = iota
	DelegateCall
)

func GenerateSafeExecTx(safeAddr, to common.Address, value *big.Int, data []byte, operation uint8, signatures []byte) {
	safeABI, err := abi.JSON(strings.NewReader(GnosisSafeABI))
	if err != nil {
		log.Panic(err)
	}
	/*
		to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte
	*/
	safeTxGas := big.NewInt(0)
	baseGas := big.NewInt(0)
	gasPrice := big.NewInt(0)
	gasToken := common.Address{}
	refundReceiver := common.Address{}
	encoded, err := safeABI.Methods["execTransaction"].Inputs.Pack(to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("encoded: %02x", encoded)
}

type MultisendSafe struct {
	conn         *ethclient.Client
	safe         *GnosisSafe
	value        *big.Int
	calldata     []byte
	to           common.Address
	safeABI      abi.ABI
	participants []common.Address
}

func NewMultisendSafe(conn *ethclient.Client, safeAddr, multisendAddr common.Address, calldata []byte, value *big.Int) (*MultisendSafe, error) {
	var err error

	ms := &MultisendSafe{}
	ms.conn = conn
	ms.safe, err = NewGnosisSafe(safeAddr, conn)
	if err != nil {
		return nil, err
	}

	safeABI, err := abi.JSON(strings.NewReader(GnosisSafeABI))
	if err != nil {
		log.Panic(err)
	}
	ms.safeABI = safeABI
	ms.to = multisendAddr
	ms.value = value
	ms.calldata = calldata

	return ms, nil
}

var (
	safeTxGas      = big.NewInt(0)
	baseGas        = big.NewInt(0)
	gasPrice       = big.NewInt(0)
	gasToken       = common.Address{}
	refundReceiver = common.Address{}
)

func (s *MultisendSafe) GetNonce() (*big.Int, error) {
	nonce, err := s.safe.Nonce(nil)
	if err != nil {
		return nil, err
	}
	return nonce, nil
}

func (s *MultisendSafe) GetSafe() *GnosisSafe {
	return s.safe
}

func (s *MultisendSafe) GetSafeTxHash(nonce int64) ([]byte, error) {
	safeTxHash, err := s.safe.EncodeTransactionData(nil,
		s.to,
		s.value,
		s.calldata,
		uint8(DelegateCall),
		safeTxGas,
		baseGas,
		gasPrice,
		gasToken,
		refundReceiver,
		big.NewInt(nonce),
	)
	if err != nil {
		return nil, err
	}
	return safeTxHash, nil
}

func (s *MultisendSafe) GetTxDataHash(nonce int64) (common.Hash, error) {
	txData, err := s.GetSafeTxHash(nonce)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(txData), nil
}

func (s *MultisendSafe) GenSafeCalldataWithoutSignature(nonce, nOfSig int64) ([]byte, error) {
	to := s.to
	value := s.value
	data := s.calldata
	signaturesPadding := make([]byte, 65*nOfSig)
	encoded, err := s.safeABI.Methods["execTransaction"].Inputs.Pack(to, value, data, uint8(DelegateCall), safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signaturesPadding)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func genApproveHashSigs(addrs []common.Address) []byte {
	approveHashSig := make([]byte, 65*len(addrs))
	for i, addr := range addrs {
		copy(approveHashSig[i*65:], addr.Hash().Bytes())
		approveHashSig[i*65+64] = 0x01
	}
	return approveHashSig
}

func (s *MultisendSafe) GenSafeCalldataWithApproveHash(participants []common.Address) ([]byte, error) {
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Hash().Big().Cmp(participants[j].Hash().Big()) == -1
	})

	to := s.to
	value := s.value
	data := s.calldata
	// spew.Dump(data)
	signatures := genApproveHashSigs(participants)
	selector := common.Hex2Bytes("6a761202")
	encoded, err := s.safeABI.Methods["execTransaction"].Inputs.Pack(to, value, data, uint8(DelegateCall), safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures)
	if err != nil {
		return nil, err
	}
	return append(selector, encoded...), nil
}

func (s *MultisendSafe) CommitApproveHash(approveHash common.Hash) {
	s.safe.ApproveHash(nil, approveHash)
}
