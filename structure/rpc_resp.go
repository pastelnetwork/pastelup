package structure

// RPCPastelID RPC result structure from pastelid newkey
type RPCPastelID struct {
	Pastelid string
}

// RPCPastelMSStatus RPC result structure from masternode status
type RPCPastelMSStatus struct {
	AssetID                int
	AssetName              string
	AssetStartTime         uint64
	Attempt                int
	IsBlockchainSynced     bool
	IsMasternodeListSynced bool
	IsWinnersListSynced    bool
	IsSynced               bool
	IsFailed               bool
}

// RPCGetInfo RPC result structure from getinfo
type RPCGetInfo struct {
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
	Errors          string  `json:"errors"`
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
