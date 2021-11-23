// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package multisend

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// MultiSendCallOnlyABI is the input ABI used to generate the binding from.
const MultiSendCallOnlyABI = "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"transactions\",\"type\":\"bytes\"}],\"name\":\"multiSend\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]"

// MultiSendCallOnly is an auto generated Go binding around an Ethereum contract.
type MultiSendCallOnly struct {
	MultiSendCallOnlyCaller     // Read-only binding to the contract
	MultiSendCallOnlyTransactor // Write-only binding to the contract
	MultiSendCallOnlyFilterer   // Log filterer for contract events
}

// MultiSendCallOnlyCaller is an auto generated read-only Go binding around an Ethereum contract.
type MultiSendCallOnlyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MultiSendCallOnlyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MultiSendCallOnlyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MultiSendCallOnlySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MultiSendCallOnlySession struct {
	Contract     *MultiSendCallOnly // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// MultiSendCallOnlyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MultiSendCallOnlyCallerSession struct {
	Contract *MultiSendCallOnlyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// MultiSendCallOnlyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MultiSendCallOnlyTransactorSession struct {
	Contract     *MultiSendCallOnlyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// MultiSendCallOnlyRaw is an auto generated low-level Go binding around an Ethereum contract.
type MultiSendCallOnlyRaw struct {
	Contract *MultiSendCallOnly // Generic contract binding to access the raw methods on
}

// MultiSendCallOnlyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MultiSendCallOnlyCallerRaw struct {
	Contract *MultiSendCallOnlyCaller // Generic read-only contract binding to access the raw methods on
}

// MultiSendCallOnlyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MultiSendCallOnlyTransactorRaw struct {
	Contract *MultiSendCallOnlyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMultiSendCallOnly creates a new instance of MultiSendCallOnly, bound to a specific deployed contract.
func NewMultiSendCallOnly(address common.Address, backend bind.ContractBackend) (*MultiSendCallOnly, error) {
	contract, err := bindMultiSendCallOnly(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnly{MultiSendCallOnlyCaller: MultiSendCallOnlyCaller{contract: contract}, MultiSendCallOnlyTransactor: MultiSendCallOnlyTransactor{contract: contract}, MultiSendCallOnlyFilterer: MultiSendCallOnlyFilterer{contract: contract}}, nil
}

// NewMultiSendCallOnlyCaller creates a new read-only instance of MultiSendCallOnly, bound to a specific deployed contract.
func NewMultiSendCallOnlyCaller(address common.Address, caller bind.ContractCaller) (*MultiSendCallOnlyCaller, error) {
	contract, err := bindMultiSendCallOnly(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyCaller{contract: contract}, nil
}

// NewMultiSendCallOnlyTransactor creates a new write-only instance of MultiSendCallOnly, bound to a specific deployed contract.
func NewMultiSendCallOnlyTransactor(address common.Address, transactor bind.ContractTransactor) (*MultiSendCallOnlyTransactor, error) {
	contract, err := bindMultiSendCallOnly(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyTransactor{contract: contract}, nil
}

// NewMultiSendCallOnlyFilterer creates a new log filterer instance of MultiSendCallOnly, bound to a specific deployed contract.
func NewMultiSendCallOnlyFilterer(address common.Address, filterer bind.ContractFilterer) (*MultiSendCallOnlyFilterer, error) {
	contract, err := bindMultiSendCallOnly(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MultiSendCallOnlyFilterer{contract: contract}, nil
}

// bindMultiSendCallOnly binds a generic wrapper to an already deployed contract.
func bindMultiSendCallOnly(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MultiSendCallOnlyABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSendCallOnly *MultiSendCallOnlyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSendCallOnly.Contract.MultiSendCallOnlyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSendCallOnly *MultiSendCallOnlyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.MultiSendCallOnlyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSendCallOnly *MultiSendCallOnlyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.MultiSendCallOnlyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MultiSendCallOnly *MultiSendCallOnlyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MultiSendCallOnly.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MultiSendCallOnly *MultiSendCallOnlyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MultiSendCallOnly *MultiSendCallOnlyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.contract.Transact(opts, method, params...)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnly *MultiSendCallOnlyTransactor) MultiSend(opts *bind.TransactOpts, transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnly.contract.Transact(opts, "multiSend", transactions)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnly *MultiSendCallOnlySession) MultiSend(transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.MultiSend(&_MultiSendCallOnly.TransactOpts, transactions)
}

// MultiSend is a paid mutator transaction binding the contract method 0x8d80ff0a.
//
// Solidity: function multiSend(bytes transactions) payable returns()
func (_MultiSendCallOnly *MultiSendCallOnlyTransactorSession) MultiSend(transactions []byte) (*types.Transaction, error) {
	return _MultiSendCallOnly.Contract.MultiSend(&_MultiSendCallOnly.TransactOpts, transactions)
}
