package app

import (
	"bytes"
	"crypto/sha256"
	"dbc-node/messages"
	"dbc-node/modules"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	tendermint "github.com/tendermint/tendermint/abci/types"
)

// TODO: ZKP in payload acceptance and maybe validation
// TODO: refactoring, better tests
// TODO: cosmos sdk integration
// TODO: ethereum integration

type DataBlockChain struct {
	Height    int64
	Proposer  []byte
	Confirmed []state // written at 2nd commit
	Committed state   // written at 1st commit
	New       state   // written at deliverTx
}

type state struct {
	Dataset *modules.Dataset
	Balance *modules.Balance
}

func (state state) hash() []byte {
	return append(state.Dataset.Hash(), state.Balance.Hash()...)
}

var _ tendermint.Application = (*DataBlockChain)(nil)

func NewDataBlockChain(genUsers, genValidators map[string]int64) *DataBlockChain {
	balance := modules.NewBalance(&modules.Balance{
		Users:      genUsers,
		Validators: genValidators,
	})
	dataset := modules.NewDataset(&modules.Dataset{}, balance)
	state := state{
		Dataset: dataset,
		Balance: balance,
	}
	return &DataBlockChain{
		Height: 0,
		New:    state,
	}
}

func (dbc *DataBlockChain) stateAtHeight(height int) state {
	if dbc.Confirmed == nil {
		return state{}
	}
	switch height {
	case 0: // return dataset at current height: last confirmed state
		return dbc.Confirmed[len(dbc.Confirmed)-1]
	case 1: // at height 1 there's no confirmed state
		return state{}
	default: // confirmed state start at height 2
		return dbc.Confirmed[height-2]
	}
}

func (dbc *DataBlockChain) Info(requestInfo tendermint.RequestInfo) tendermint.ResponseInfo {
	responseInfo := tendermint.ResponseInfo{
		Data:             "Some arbitrary information about dbc-node app",
		Version:          "V1",
		AppVersion:       1,
		LastBlockHeight:  dbc.Height,
		LastBlockAppHash: dbc.Committed.hash(),
	}
	return responseInfo
}

func (dbc *DataBlockChain) SetOption(requestSetOption tendermint.RequestSetOption) tendermint.ResponseSetOption {
	responseSetOption := tendermint.ResponseSetOption{
		Code: 0,
		Log:  "",
		Info: "",
	}
	return responseSetOption
}

func (dbc *DataBlockChain) Query(requestQuery tendermint.RequestQuery) tendermint.ResponseQuery {
	data := make([]byte, base64.StdEncoding.DecodedLen(len(requestQuery.Data)))
	_, _ = base64.StdEncoding.Decode(data, requestQuery.Data)
	data = bytes.Trim(data, "\x00")
	var query messages.Query
	_ = json.Unmarshal(data, &query)
	var value []byte
	state := dbc.stateAtHeight(int(requestQuery.Height))
	switch query.QrType {
	case messages.QueryDataset:
		value, _ = json.Marshal(state.Dataset)
	case messages.QueryData:
		value, _ = json.Marshal(state.Dataset.DataList[query.DataIndex])
	case messages.QueryVersion:
		value, _ = json.Marshal(state.Dataset.DataList[query.DataIndex].VersionList[query.VersionIndex])
	case messages.QueryDescription:
		value, _ = json.Marshal(state.Dataset.DataList[query.DataIndex].Description)
	case messages.QueryValidation:
		value, _ = json.Marshal(state.Dataset.DataList[query.DataIndex].VersionList[query.VersionIndex].Validation)
	case messages.QueryPayload:
		value, _ = json.Marshal(state.Dataset.DataList[query.DataIndex].VersionList[query.VersionIndex].Payload)
	case messages.QueryAcceptedPayload:
		value, _ = json.Marshal(state.Dataset.DataList[query.DataIndex].VersionList[query.VersionIndex].AcceptedPayload)
	case messages.QueryBalance:
		value, _ = json.Marshal(state.Balance.Users)
	case messages.QueryStake:
		value, _ = json.Marshal(state.Balance.Validators)
	}
	responseQuery := tendermint.ResponseQuery{
		Code:      uint32(0),
		Log:       "",
		Info:      "",
		Index:     -1,
		Key:       requestQuery.Data,
		Value:     value,
		Proof:     nil,
		Height:    0,
		Codespace: "",
	}
	return responseQuery
}

func (dbc *DataBlockChain) CheckTx(requestCheckTx tendermint.RequestCheckTx) tendermint.ResponseCheckTx {
	responseCheckTx := tendermint.ResponseCheckTx{
		Code:      uint32(0),
		Data:      nil,
		Log:       "",
		Info:      "",
		GasWanted: 0,
		GasUsed:   0,
		Events:    nil,
		Codespace: "",
	}
	return responseCheckTx
}

func (dbc *DataBlockChain) InitChain(requestInitChain tendermint.RequestInitChain) tendermint.ResponseInitChain {
	responseInitChain := tendermint.ResponseInitChain{
		ConsensusParams: nil,
		Validators:      nil,
	}
	return responseInitChain
}

func (dbc *DataBlockChain) BeginBlock(requestBeginBlock tendermint.RequestBeginBlock) tendermint.ResponseBeginBlock {
	dbc.Proposer = requestBeginBlock.Header.ProposerAddress
	responseBeginBlock := tendermint.ResponseBeginBlock{
		Events: nil,
	}
	return responseBeginBlock
}

func (dbc *DataBlockChain) DeliverTx(requestDeliverTx tendermint.RequestDeliverTx) tendermint.ResponseDeliverTx {
	tx := make([]byte, base64.StdEncoding.DecodedLen(len(requestDeliverTx.Tx)))
	_, _ = base64.StdEncoding.Decode(tx, requestDeliverTx.Tx)
	tx = bytes.Trim(tx, "\x00")
	var transaction messages.Transaction
	_ = json.Unmarshal(tx, &transaction)
	txHash := sha256.Sum256(tx)
	fee := &modules.Fee{
		ValAddr: dbc.Proposer,
		TxHash:  txHash[:],
	}
	var feeErr error
	var txErr error
	switch transaction.TxType {
	case messages.TxAddData:
		description := transaction.Description
		fee.User = description.Requirer
		if feeErr = dbc.New.Balance.AddFee(fee); feeErr == nil {
			txErr = dbc.New.Dataset.AddData(description)
		}
	case messages.TxAddValidation:
		validation := transaction.Validation
		fee.User = validation.ValidatorAddr
		if feeErr := dbc.New.Balance.AddFee(fee); feeErr == nil {
			txErr = dbc.New.Dataset.AddValidation(validation, transaction.DataIndex)
		}
	case messages.TxAddPayload:
		payload := transaction.Payload
		fee.User = payload.ProviderAddr
		if feeErr := dbc.New.Balance.AddFee(fee); feeErr == nil {
			txErr = dbc.New.Dataset.AddPayload(payload, transaction.DataIndex, transaction.VersionIndex)
		}
	case messages.TxAcceptPayload:
		acceptedPayload := transaction.AcceptedPayload
		fee.User = acceptedPayload.AcceptorAddr
		if feeErr := dbc.New.Balance.AddFee(fee); feeErr == nil {
			txErr = dbc.New.Dataset.AcceptPayload(acceptedPayload, transaction.DataIndex, transaction.VersionIndex)
		}
	case messages.TxTransfer:
		transfer := transaction.Transfer
		fee.User = transfer.Sender
		if feeErr := dbc.New.Balance.AddFee(fee); feeErr == nil {
			txErr = dbc.New.Balance.AddTransfer(transfer)
		}
	case messages.TxStake:
		stake := transaction.Stake
		fee.User = stake.User
		if feeErr := dbc.New.Balance.AddFee(fee); feeErr == nil {
			txErr = dbc.New.Balance.AddStake(stake)
		}
	}
	code := uint32(0)
	feedback := "transaction delivered successfully"
	if feeErr != nil {
		code = 1
		feedback = feeErr.Error()
	} else if txErr != nil {
		code = 1
		feedback = txErr.Error()
	}
	responseDeliverTx := tendermint.ResponseDeliverTx{
		Code:      code,
		Data:      nil,
		Log:       feedback,
		Info:      feedback,
		GasWanted: 0,
		GasUsed:   0,
		Events:    nil,
		Codespace: "",
	}
	return responseDeliverTx
}

func (dbc *DataBlockChain) EndBlock(requestEndBlock tendermint.RequestEndBlock) tendermint.ResponseEndBlock {
	validatorUpdates := tendermint.ValidatorUpdates{}
	for validator, _ := range dbc.New.Balance.ValChanges {
		stake := dbc.New.Balance.Validators[validator]
		validatorBytes, _ := hex.DecodeString(validator)
		validatorUpdate := tendermint.Ed25519ValidatorUpdate(validatorBytes, stake)
		validatorUpdates = append(validatorUpdates, validatorUpdate)
	}
	responseEndBlock := tendermint.ResponseEndBlock{
		ValidatorUpdates:      validatorUpdates,
		ConsensusParamUpdates: nil,
		Events:                nil,
	}
	return responseEndBlock
}

func (dbc *DataBlockChain) Commit() tendermint.ResponseCommit {
	if dbc.Height > 0 { // we don't append to confirmed in the first commit, since there's no committed state yet
		dbc.Confirmed = append(dbc.Confirmed, dbc.Committed)
	}
	dbc.Committed = dbc.New
	balance := modules.NewBalance(dbc.Committed.Balance)
	dataset := modules.NewDataset(dbc.Committed.Dataset, balance)
	dbc.New = state{
		Dataset: dataset,
		Balance: balance,
	}
	dbc.Height++
	responseCommit := tendermint.ResponseCommit{
		Data:         dbc.Committed.hash(),
		RetainHeight: 0,
	}
	return responseCommit
}
