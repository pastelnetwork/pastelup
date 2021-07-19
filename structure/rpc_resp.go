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
