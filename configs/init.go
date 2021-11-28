package configs

// Init contains config of the Init command
type Init struct {
	WorkingDir    string `json:"workdir,omitempty"`
	Network       string `json:"network,omitempty"`
	RPCPort       int    `json:"rpc-port,omitempty"`
	RPCUser       string `json:"rpc-user,omitempty"`
	RPCPwd        string `json:"rpc-pwd,omitempty"`
	Force         bool   `json:"force,omitempty"`
	Peers         string `json:"peers"`
	PastelExecDir string `json:"pastelexecdir,omitempty"`
	Version       string `json:"nodeversion,omitempty"`
	StartedRemote bool   `json:"started-remote,omitempty"`
	EnableService bool   `json:"enableservice,omitempty"`
	UserPw        string `json:"user-pw,omitempty"`

	// Configs for remote session
	RemoteWorkingDir       string `json:"remoteworkingdir,omitempty"`
	RemotePastelExecDir    string `json:"remotepastelexecdir,omitempty"`
	RemotePastelUtilityDir string `json:"remotepastelutilitydir,omitempty"`
	// RemoteIP               string `json:"remote-ip"`
	// RemotePort             int    `json:"remote-port"`
	// RemoteUser             string `json:"remote-user"`
	// SSHKey                 string `json:"ssh-key"`

	BinUtilityPath   string `json:"utility-path,omitempty"`
	BinComponentPath string `json:"bin-path,omitempty"`
}

/*
// NewInit returns a new Init instance.
func NewInit() *Init {
	return &Init{}
}*/
