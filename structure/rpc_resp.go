package structure

import (
	"encoding/json"
	"fmt"
)

// RPCPastelID RPC result structure from pastelid newkey
type RPCPastelID struct {
	Pastelid string
}

// RPCPastelMNStatus is the RPC result structure from masternode status
type RPCPastelMNStatus struct {
	Result MNStatusResult `json:"result"`
	Error  struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
}

// MNStatusResult is the result field for the RPC command response
type MNStatusResult struct {
	Outpoint string `json:"outpoint,omitempty"`
	Service  string `json:"service,omitempty"`
	Status   string `json:"status,omitempty"`
}

// String returns the struct as a string
func (s RPCPastelMNStatus) String() string {
	return toString(s)
}

// String returns the struct as a string
func (s MNStatusResult) String() string {
	return toString(s)
}

// RPCPastelMNSyncStatus RPC result structure from masternode sync status
type RPCPastelMNSyncStatus struct {
	Result MNSyncStatusResult `json:"result"`
	Error  struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
}

// MNSyncStatusResult is the result field for the RPC command response
type MNSyncStatusResult struct {
	AssetID                int    `json:"AssetID,omitempty"`
	AssetName              string `json:"AssetName,omitempty"`
	AssetStartTime         uint64 `json:"AssetStartTime,omitempty"`
	Attempt                int    `json:"Attempt,omitempty"`
	IsBlockchainSynced     bool   `json:"IsBlockchainSynced,omitempty"`
	IsMasternodeListSynced bool   `json:"IsMasternodeListSynced,omitempty"`
	IsWinnersListSynced    bool   `json:"IsWinnersListSynced,omitempty"`
	IsSynced               bool   `json:"IsSynced,omitempty"`
	IsFailed               bool   `json:"IsFailed,omitempty"`
}

// String returns the struct as a string
func (s RPCPastelMNSyncStatus) String() string {
	return toString(s)
}

// String returns the struct as a string
func (s MNSyncStatusResult) String() string {
	return toString(s)
}

// RPCGetInfo RPC result structure from getinfo
type RPCGetInfo struct {
	Result GetInfoResult `json:"result"`
	Error  interface{}   `json:"error"`
}

// GetInfoResult is the result field for the RPC command response
type GetInfoResult struct {
	Version         int     `json:"version"`
	Protocolversion int     `json:"protocolversion"`
	Walletversion   int     `json:"walletversion"`
	Balance         float64 `json:"balance"`
	Blocks          int     `json:"blocks"`
	Timeoffset      int     `json:"timeoffset"`
	Connections     int     `json:"connections"`
	Proxy           string  `json:"proxy"`
	Difficulty      float64 `json:"difficulty"`
	Chain           string  `json:"chain"`
	Keypoololdest   int     `json:"keypoololdest"`
	Keypoolsize     int     `json:"keypoolsize"`
	Paytxfee        float64 `json:"paytxfee"`
	Relayfee        float64 `json:"relayfee"`
}

// String returns the struct as a string
func (s RPCGetInfo) String() string {
	return toString(s)
}

// String returns the struct as a string
func (s GetInfoResult) String() string {
	return toString(s)
}

// RPCMasternodeConf RPC result structure from masterode list-conf
type RPCMasternodeConf struct {
	Result MasternodeConfResult `json:"result"`
	Error  interface{}          `json:"error"`
}

// MasternodeConfResult is the result field for the RPC command response
type MasternodeConfResult struct {
	Masternode struct {
		Alias       string `json:"alias"`
		Address     string `json:"address"`
		PrivateKey  string `json:"privateKey"`
		TxHash      string `json:"txHash"`
		OutputIndex string `json:"outputIndex"`
		ExtAddress  string `json:"extAddress"`
		ExtP2P      string `json:"extP2P"`
		ExtKey      string `json:"extKey"`
		ExtCfg      string `json:"extCfg"`
		Status      string `json:"status"`
	} `json:"masternode"`
}

// String returns the struct as a string
func (s RPCMasternodeConf) String() string {
	return toString(s)
}

// String returns the struct as a string
func (s MasternodeConfResult) String() string {
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
