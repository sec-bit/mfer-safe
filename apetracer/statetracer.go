package apetracer

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func NewStateTracer() *StateTracer {
	return &StateTracer{
		slotAccessed: make(map[common.Address]map[common.Hash]common.Hash),
	}
}

type StateTracer struct {
	slotAccessed map[common.Address]map[common.Hash]common.Hash
}

func (tracer *StateTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

func (tracer *StateTracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	contract := scope.Contract.Address()
	if _, ok := tracer.slotAccessed[contract]; !ok {
		tracer.slotAccessed[contract] = make(map[common.Hash]common.Hash)
	}
	switch op {
	case vm.SLOAD:
		{
			stack := scope.Stack.Data()
			key := stack[len(stack)-1]
			keyHash := common.Hash(key.Bytes32())
			val := env.StateDB.GetState(scope.Contract.Address(), keyHash)
			tracer.slotAccessed[contract][keyHash] = val
		}
	case vm.SSTORE:
		{
			stack := scope.Stack.Data()
			key, value := stack[len(stack)-1], stack[len(stack)-2]
			keyHash := common.Hash(key.Bytes32())
			valHash := common.Hash(value.Bytes32())
			tracer.slotAccessed[contract][keyHash] = valHash
		}
	}
}
func (tracer *StateTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}
func (tracer *StateTracer) CaptureExit(output []byte, gasUsed uint64, err error) {}
func (tracer *StateTracer) CaptureFault(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}
func (tracer *StateTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {}

func (tracer *StateTracer) GetResult() map[common.Address]map[common.Hash]common.Hash {
	return tracer.slotAccessed
}

func (tracer *StateTracer) Reset() {
	tracer.slotAccessed = make(map[common.Address]map[common.Hash]common.Hash)
}
