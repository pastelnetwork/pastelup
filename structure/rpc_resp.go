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
