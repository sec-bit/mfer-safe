package apetracer

import (
	"hash"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

type KeccakTracer struct{}

func (tracer *KeccakTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

func (tracer *KeccakTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
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
			contract := scope.Contract.Address()
			callData := scope.Contract.Input

			log.Printf("Keccak Detected\nContract: %s\ncalldata: 0x%02x\npreimage: %02x\nhash: %s", contract.Hex(), callData, data, hasherBuf.Hex())
		}
	case vm.ORIGIN:
		{
			contract := scope.Contract.Address()
			log.Printf("Contract: %s uses tx.origin", contract.Hex())
		}
	}
}
func (tracer *KeccakTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}
func (tracer *KeccakTracer) CaptureExit(output []byte, gasUsed uint64, err error) {}
func (tracer *KeccakTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}
func (tracer *KeccakTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {}
