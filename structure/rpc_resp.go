package structure

// RPC_PastelID RPC result from pastelid newkey 
type RPCPastelID struct {
	Pastelid string
}

// RPC_PastelMSStatus RPC result from masternode status 
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
