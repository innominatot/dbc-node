package app

import (
	"bytes"
	"dbc-node/messages"
	"dbc-node/statedt"
	"encoding/base64"
	"encoding/json"
	tendermint "github.com/tendermint/tendermint/abci/types"
)

// TODO: stake & unstake transactions and fees (stake economy)
// TODO: consensus level payload validation (see: Whitepaper, Multiple Choice Problem)

type dataBlockChain struct {
	Height    int64
	Confirmed []*statedt.State // written at 2nd commit
	Committed *statedt.State   // written at 1st commit
	New       *statedt.State   // written at deliverTx
}

var _ tendermint.Application = (*dataBlockChain)(nil)

func NewDataBlockChain() *dataBlockChain {
	state := statedt.NewState(&statedt.State{})
	return &dataBlockChain{
		Height: 0,
		New:    state,
	}
}

func (dbc *dataBlockChain) stateAtHeight(height int) *statedt.State {
	if dbc.Confirmed == nil {
		return nil
	}
	switch height {
	case 0: // return state at current height: last confirmed state
		return dbc.Confirmed[len(dbc.Confirmed)-1]
	case 1: // at height 1 there's no confirmed state
		return nil
	default: // confirmed states start at height 2
		return dbc.Confirmed[height-2]
	}
}

func (dbc *dataBlockChain) Info(requestInfo tendermint.RequestInfo) tendermint.ResponseInfo {
	responseInfo := tendermint.ResponseInfo{
		Data:             "Some arbitrary information about dbc-node app",
		Version:          "V1",
		AppVersion:       1,
		LastBlockHeight:  dbc.Height,
		LastBlockAppHash: dbc.Committed.Hash(),
	}
	return responseInfo
}

func (dbc *dataBlockChain) SetOption(requestSetOption tendermint.RequestSetOption) tendermint.ResponseSetOption {
	responseSetOption := tendermint.ResponseSetOption{
		Code: 0,
		Log:  "",
		Info: "",
	}
	return responseSetOption
}

func (dbc *dataBlockChain) Query(requestQuery tendermint.RequestQuery) tendermint.ResponseQuery {
	data := make([]byte, base64.StdEncoding.DecodedLen(len(requestQuery.Data)))
	_, _ = base64.StdEncoding.Decode(data, requestQuery.Data)
	data = bytes.Trim(data, "\x00")
	var query messages.Query
	_ = json.Unmarshal(data, &query)
	var value []byte
	state := dbc.stateAtHeight(int(requestQuery.Height))
	switch query.QrType {
	case messages.QueryState:
		value, _ = json.Marshal(state)
	case messages.QueryData:
		value, _ = json.Marshal(state.DataList[query.DataIndex])
	case messages.QueryVersion:
		value, _ = json.Marshal(state.DataList[query.DataIndex].VersionList[query.VersionIndex])
	case messages.QueryDescription:
		value, _ = json.Marshal(state.DataList[query.DataIndex].Description)
	case messages.QueryValidation:
		value, _ = json.Marshal(state.DataList[query.DataIndex].VersionList[query.VersionIndex].Validation)
	case messages.QueryPayload:
		value, _ = json.Marshal(state.DataList[query.DataIndex].VersionList[query.VersionIndex].Payload)
	case messages.QueryAcceptedPayload:
		value, _ = json.Marshal(state.DataList[query.DataIndex].VersionList[query.VersionIndex].AcceptedPayload)
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

func (dbc *dataBlockChain) CheckTx(requestCheckTx tendermint.RequestCheckTx) tendermint.ResponseCheckTx {
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

func (dbc *dataBlockChain) InitChain(requestInitChain tendermint.RequestInitChain) tendermint.ResponseInitChain {
	responseInitChain := tendermint.ResponseInitChain{
		ConsensusParams: nil,
		Validators:      nil,
	}
	return responseInitChain
}

func (dbc *dataBlockChain) BeginBlock(requestBeginBlock tendermint.RequestBeginBlock) tendermint.ResponseBeginBlock {
	responseBeginBlock := tendermint.ResponseBeginBlock{
		Events: nil,
	}
	return responseBeginBlock
}

func (dbc *dataBlockChain) DeliverTx(requestDeliverTx tendermint.RequestDeliverTx) tendermint.ResponseDeliverTx {
	tx := make([]byte, base64.StdEncoding.DecodedLen(len(requestDeliverTx.Tx)))
	_, _ = base64.StdEncoding.Decode(tx, requestDeliverTx.Tx)
	tx = bytes.Trim(tx, "\x00")
	var transaction messages.Transaction
	_ = json.Unmarshal(tx, &transaction)
	switch transaction.TxType {
	case messages.TxAddData:
		description := transaction.Description
		dbc.New.AddData(description)
	case messages.TxAddValidation:
		validation := transaction.Validation
		dbc.New.AddValidation(validation, transaction.DataIndex)
	case messages.TxAddPayload:
		payload := transaction.Payload
		dbc.New.AddPayload(payload, transaction.DataIndex, transaction.VersionIndex)
	case messages.TxAcceptPayload:
		acceptedPayload := transaction.AcceptedPayload
		dbc.New.AcceptPayload(acceptedPayload, transaction.DataIndex, transaction.VersionIndex)
	}
	responseDeliverTx := tendermint.ResponseDeliverTx{
		Code:      uint32(0),
		Data:      nil,
		Log:       "",
		Info:      "",
		GasWanted: 0,
		GasUsed:   0,
		Events:    nil,
		Codespace: "",
	}
	return responseDeliverTx
}

func (dbc *dataBlockChain) EndBlock(requestEndBlock tendermint.RequestEndBlock) tendermint.ResponseEndBlock {
	responseEndBlock := tendermint.ResponseEndBlock{
		ValidatorUpdates:      nil,
		ConsensusParamUpdates: nil,
		Events:                nil,
	}
	return responseEndBlock
}

func (dbc *dataBlockChain) Commit() tendermint.ResponseCommit {
	if dbc.Height > 0 { // we don't append to confirmed in the first commit, since there's no committed state yet
		dbc.Confirmed = append(dbc.Confirmed, dbc.Committed)
	}
	dbc.Committed = dbc.New
	dbc.New = statedt.NewState(dbc.Committed)
	dbc.Height++
	responseCommit := tendermint.ResponseCommit{
		Data:         dbc.Committed.Hash(),
		RetainHeight: 0,
	}
	return responseCommit
}