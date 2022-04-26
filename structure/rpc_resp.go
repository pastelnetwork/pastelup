package structure

import (
	"encoding/json"
	"fmt"
)

// RPCPastelID RPC result structure from pastelid newkey
type RPCPastelID struct {
	Pastelid string
}

// RPCPastelMSStatus RPC result structure from masternode status
type RPCPastelMSStatus struct {
	AssetID                int    `json:"AssetID,omitempty"`
	AssetName              string `json:"AssetName,omitempty"`
	AssetStartTime         uint64 `json:"AssetStartTime,omitempty"`
	Attempt                int    `json:"Attempt,omitempty"`
	IsBlockchainSynced     bool   `json:"IsBlockchainSynced,omitempty"`
	IsMasternodeListSynced bool   `json:"IsMasternodeListSynced,omitempty"`
	IsWinnersListSynced    bool   `json:"IsWinnersListSynced,omitempty"`
	IsSynced               bool   `json:"IsSynced,omitempty"`
	IsFailed               bool   `json:"IsFailed,omitempty"`
	Error                  struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
}

func (s RPCPastelMSStatus) String() string {
	return toString(s)
}

// RPCGetInfo RPC result structure from getinfo
type RPCGetInfo struct {
	Result struct {
		Version         int     `json:"version"`
		Protocolversion int     `json:"protocolversion"`
		Walletversion   int     `json:"walletversion"`
		Balance         float64 `json:"balance"`
		Blocks          int     `json:"blocks"`
		Timeoffset      int     `json:"timeoffset"`
		Connections     int     `json:"connections"`
		Proxy           string  `json:"proxy"`
		Difficulty      float64 `json:"difficulty"`
		Testnet         bool    `json:"testnet"`
		Keypoololdest   int     `json:"keypoololdest"`
		Keypoolsize     int     `json:"keypoolsize"`
		Paytxfee        float64 `json:"paytxfee"`
		Relayfee        float64 `json:"relayfee"`
	}
	Error string `json:"error"`
}

func (s RPCGetInfo) String() string {
	return toString(s)
}

// TxInfo Transaction information
type TxInfo struct {
	Account         string
	Address         string
	Category        string
	Amount          float64
	Vout            uint64
	Confirmation    uint64
	BlockHash       string
	BlockIndex      uint64
	BlockTime       uint64
	Expiryheight    uint64
	TxID            string
	WalletConflicts []string
	Time            uint64
	TimeReceived    uint64
	Vjoinsplit      []string
	Size            uint64
}

func toString(s interface{}) string {
	b, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return fmt.Sprintf("%+v", s)
	}
	return string(b)
}
