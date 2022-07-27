package mfertracer

import (
	"hash"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/crypto/sha3"
)

/*
type Tracer interface {
	CaptureStart(env *EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int)
	CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error)
	CaptureEnter(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	CaptureExit(output []byte, gasUsed uint64, err error)
	CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, depth int, err error)
	CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error)
}
*/

type KeccakOp struct {
	Contract common.Address
	Preimage hexutil.Bytes
	Hash     common.Hash
}

type StorageOp struct {
	Contract common.Address
	Key      common.Hash
	Value    common.Hash
	IsWrite  bool
}

type GasOp struct {
	Contract  common.Address
	GasRemain uint64
}

func NewKeccakTracer() *KeccakTracer {
	return &KeccakTracer{
		traceOps: make([]interface{}, 0),
	}
}

type KeccakTracer struct {
	traceOps []interface{}
}

func (tracer *KeccakTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

func (tracer *KeccakTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	contract := scope.Contract.Address()
	switch op {
	case vm.SHA3:
		{
			// st.data[len(st.data)-1]
			// offset, size := scope.Stack.pop(), scope.Stack.peek()
			// data := scope.Memory.GetPtr(int64(offset.Uint64()), int64(size.Uint64()))

			stack := scope.Stack.Data()
			offset, size := stack[len(stack)-1], stack[len(stack)-2]
			data := scope.Memory.GetPtr(int64(offset.Uint64()), int64(size.Uint64()))
			hasher := sha3.NewLegacyKeccak256().(keccakState)
			hasher.Write(data)

			var hasherBuf common.Hash
			hasher.Read(hasherBuf[:])

			// evm := interpreter.evm
			// if evm.Config.EnablePreimageRecording {
			// 	evm.StateDB.AddPreimage(interpreter.hasherBuf, data)
			// }
			// callData := scope.Contract.Input
			keccakOp := KeccakOp{
				Contract: contract,
				Preimage: data,
				Hash:     hasherBuf,
			}
			tracer.traceOps = append(tracer.traceOps, keccakOp)
			// log.Printf("Keccak Detected\nContract: %s\ncalldata: 0x%02x\npreimage: %02x\nhash: %s", contract.Hex(), callData, data, hasherBuf.Hex())
		}
	case vm.ORIGIN:
		{
			log.Printf("Contract: %s uses tx.origin", contract.Hex())
		}
	case vm.REVERT:
		{
			stack := scope.Stack.Data()
			offset, size := stack[len(stack)-1], stack[len(stack)-2]
			data := scope.Memory.GetPtr(int64(offset.Uint64()), int64(size.Uint64()))
			log.Printf("Contract: %s revert: 0x%02x(%s)", contract.Hex(), data, string(data))
		}
	case vm.SLOAD:
		{
			stack := scope.Stack.Data()
			key := stack[len(stack)-1]
			keyHash := common.Hash(key.Bytes32())
			val := env.StateDB.GetState(scope.Contract.Address(), keyHash)
			storageOp := StorageOp{
				Contract: contract,
				Key:      keyHash,
				Value:    val,
				IsWrite:  false,
			}
			tracer.traceOps = append(tracer.traceOps, storageOp)
		}
	case vm.SSTORE:
		{
			stack := scope.Stack.Data()
			key, value := stack[len(stack)-1], stack[len(stack)-2]
			keyHash := common.Hash(key.Bytes32())
			valHash := common.Hash(value.Bytes32())
			storageOp := StorageOp{
				Contract: contract,
				Key:      keyHash,
				Value:    valHash,
				IsWrite:  true,
			}
			tracer.traceOps = append(tracer.traceOps, storageOp)
		}
	case vm.GAS:
		{
			gasOp := GasOp{
				Contract:  contract,
				GasRemain: scope.Contract.Gas,
			}
			tracer.traceOps = append(tracer.traceOps, gasOp)
		}
	}
}
func (tracer *KeccakTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}
func (tracer *KeccakTracer) CaptureExit(output []byte, gasUsed uint64, err error) {}
func (tracer *KeccakTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}
func (tracer *KeccakTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {}

func (tracer *KeccakTracer) GetResult() []interface{} {
	return tracer.traceOps
}

func (tracer *KeccakTracer) Reset() {
	tracer.traceOps = make([]interface{}, 0)
}
