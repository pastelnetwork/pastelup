package configs

// Init contains config of the Init command
type Init struct {
	WorkingDir    string `json:"workdir,omitempty"`
	Network       string `json:"network,omitempty"`
	RPCPort       int    `json:"rpc-port,omitempty"`
	RPCUser       string `json:"rpc-user,omitempty"`
	RPCPwd        string `json:"rpc-pwd,omitempty"`
	Force         bool   `json:"force,omitempty"`
	Clean         bool   `json:"clean,omitempty"`
	Peers         string `json:"peers"`
	PastelExecDir string `json:"pastelexecdir,omitempty"`
	Version       string `json:"nodeversion,omitempty"`
	StartedRemote bool   `json:"started-remote,omitempty"`
	EnableService bool   `json:"enableservice,omitempty"`
	UserPw        string `json:"user-pw,omitempty"`
	OpMode        string `json:"opmode,omitempty"`
	ArchiveDir    string `json:"archivedir,omitempty"`
	Legacy        bool   `json:"legacy,omitempty"`
	ReIndex       bool   `json:"reindex,omitempty"`
	NoCache       bool   `json:"nocache,omitempty"`
	NoBackup      bool   `json:"nobackup,omitempty"`
	RegenRPC      bool   `json:"regen-rpc,omitempty"`
	IsTestnet     bool   `json:"isTestnet,omitempty"` // if true, pastel.conf had testnet=1 when cmd was invoked

	// Configs for remote session
	RemoteHotHomeDir       string `json:"remotehomedir,omitempty"`
	RemoteHotWorkingDir    string `json:"remoteworkingdir,omitempty"`
	RemoteHotPastelExecDir string `json:"remotepastelexecdir,omitempty"`
	RemoteIP               string `json:"remote-ip,omitempty"`
	RemotePort             int    `json:"remote-port,omitempty"`
	RemoteUser             string `json:"remote-user,omitempty"`
	RemoteSSHKey           string `json:"remote-ssh-key,omitempty"`
	InventoryFile          string `json:"inventory-file,omitempty"`
}

/*
// NewInit returns a new Init instance.
func NewInit() *Init {
	return &Init{}
}*/
