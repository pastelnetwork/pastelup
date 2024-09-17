package configs

// Init contains config of the Init command
type Init struct {
	WorkingDir                  string `json:"workdir,omitempty"`
	Network                     string `json:"network,omitempty"`
	RPCPort                     int    `json:"rpc-port,omitempty"`
	RPCUser                     string `json:"rpc-user,omitempty"`
	RPCPwd                      string `json:"rpc-pwd,omitempty"`
	Force                       bool   `json:"force,omitempty"`
	SkipSystemUpdate            bool   `json:"skip-system-update,omitempty"`
	SkipDDPackagesUpdate        bool   `json:"skip-dd-packages-update,omitempty"`
	SkipDDSupportingFilesUpdate bool   `json:"skip-dd-supporting-files-update,omitempty"`
	Clean                       bool   `json:"clean,omitempty"`
	Peers                       string `json:"peers"`
	PastelExecDir               string `json:"pastelexecdir,omitempty"`
	Version                     string `json:"nodeversion,omitempty"`
	StartedRemote               bool   `json:"started-remote,omitempty"`
	UserPw                      string `json:"user-pw,omitempty"`
	OpMode                      string `json:"opmode,omitempty"`
	ArchiveDir                  string `json:"archivedir,omitempty"`
	Legacy                      bool   `json:"legacy,omitempty"`
	ReIndex                     bool   `json:"reindex,omitempty"`
	TxIndex                     int    `json:"txindex,omitempty"`
	NoCache                     bool   `json:"nocache,omitempty"`
	NoBackup                    bool   `json:"nobackup,omitempty"`
	BackupAll                   bool   `json:"backupall,omitempty"`
	RegenRPC                    bool   `json:"regen-rpc,omitempty"`
	IsTestnet                   bool   `json:"isTestnet,omitempty"` // if true, pastel.conf had testnet=1 when cmd was invoked
	IsDevnet                    bool   `json:"isDevnet,omitempty"`  // if true, pastel.conf had devnet=1 when cmd was invoked
	EnableService               bool   `json:"enable,omitempty"`
	StartService                bool   `json:"start,omitempty"`
	ServiceTool                 string `json:"tool,omitempty"`
	ServiceSolution             string `json:"solution,omitempty"`
	DevMode                     bool   `json:"devmode,omitempty"`
	UseSnapshot                 bool   `json:"use-snapshot,omitempty"`
	SnapshotName                string `json:"snapshot-name,omitempty"`

	NodeExtIP string `json:"nodeextip,omitempty"`

	ActivateMasterNode      bool   `json:"activatemasternode,omitempty"`
	MasterNodeName          string `json:"masternodename,omitempty"`
	CreateNewMasterNodeConf bool   `json:"createnewmasternodeconf,omitempty"`
	AddToMasterNodeConf     bool   `json:"addtomasternodeconf,omitempty"`
	MasterNodeTxID          string `json:"masternodetxid,omitempty"`
	MasterNodeTxInd         string `json:"masternodetxind,omitempty"`
	DontCheckCollateral     bool   `json:"dontcheckcollateral,omitempty"`
	DontUseReindex          bool   `json:"dontusereindex,omitempty"`
	MasterNodePort          int    `json:"masternodeport,omitempty"`
	MasterNodePrivateKey    string `json:"masternodeprivatekey,omitempty"`
	MasterNodePastelID      string `json:"masternodepastelid,omitempty"`
	MasterNodePassPhrase    string `json:"masternodepassphrase,omitempty"`
	MasterNodeRPCIP         string `json:"masternoderpcip,omitempty"`
	MasterNodeRPCPort       int    `json:"masternoderpcport,omitempty"`
	MasterNodeP2PIP         string `json:"masternodep2pip,omitempty"`
	MasterNodeP2PPort       int    `json:"masternodep2pport,omitempty"`

	// Configs for remote session
	RemoteHotHomeDir       string `json:"remotehomedir,omitempty"`
	RemoteHotWorkingDir    string `json:"remoteworkingdir,omitempty"`
	RemoteHotPastelExecDir string `json:"remotepastelexecdir,omitempty"`
	RemoteIP               string `json:"remote-ip,omitempty"`
	RemotePort             int    `json:"remote-port,omitempty"`
	RemoteUser             string `json:"remote-user,omitempty"`
	RemoteSSHKey           string `json:"remote-ssh-key,omitempty"`
	InventoryFile          string `json:"inventory-file,omitempty"`
	InventoryFilter        string `json:"inventory-filter,omitempty"`
	AsyncRemote            bool   `json:"async_remote,omitempty"`
}

/*
// NewInit returns a new Init instance.
func NewInit() *Init {
	return &Init{}
}*/
