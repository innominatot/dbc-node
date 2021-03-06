package messages

import (
	"dbc-node/modules"
)

type TransactionType string

const (
	TxAddData       TransactionType = "TxAddData"
	TxAddValidation TransactionType = "TxAddValidation"
	TxAddPayload    TransactionType = "TxAddPayload"
	TxAcceptPayload TransactionType = "TxAcceptPayload"
	TxTransfer      TransactionType = "TxTransfer"
	TxStake         TransactionType = "TxStake"
)

type Transaction struct {
	TxType TransactionType

	Description     *modules.Description
	Validation      *modules.Validation
	Payload         *modules.Payload
	AcceptedPayload *modules.AcceptedPayload
	Transfer        *modules.Transfer
	Stake           *modules.Stake

	DataIndex    int
	VersionIndex int
}

type QueryType string

const (
	QueryDataset         QueryType = "QueryDataset"
	QueryData            QueryType = "QueryData"
	QueryVersion         QueryType = "QueryVersion"
	QueryDescription     QueryType = "QueryDescription"
	QueryValidation      QueryType = "QueryValidation"
	QueryPayload         QueryType = "QueryPayload"
	QueryAcceptedPayload QueryType = "QueryAcceptedPayload"
	QueryBalance         QueryType = "QueryBalance"
	QueryStake           QueryType = "QueryStake"
)

type Query struct {
	QrType       QueryType
	DataIndex    int
	VersionIndex int
}
