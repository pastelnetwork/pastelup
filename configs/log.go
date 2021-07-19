package configs

const (
	defaultLogLevel = "info"
)

// Main contains main config of the App
type Main struct {
	LogLevel string `json:"log-level,omitempty"`
	LogFile  string `json:"log-file,omitempty"`
	Quiet    bool   `json:"quiet"`
}

// NewMain returns a new Main instance.
func NewMain() *Main {
	return &Main{
		LogLevel: defaultLogLevel,
	}
}
