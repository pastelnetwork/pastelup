package structure

// declaring a struct
type RPC_PastelID struct {
	Pastelid string
}

type RPC_PastelMSStatus struct {
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
