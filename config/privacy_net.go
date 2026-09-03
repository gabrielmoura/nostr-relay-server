package config

// PrivacyConfig defines the per-network privacy/gateway settings for exposing
// the relay over Tor (onion), I2P (eepsite) and Yggdrasil (mesh) networks.
//
// The whole block is opt-in: nothing is enabled until `privacy.enabled` is
// true. Each network can run in one of two modes:
//
//	"native"   - run the network in-process (pure Go, no external daemon).
//	"external" - connect to an already-running daemon/proxy (recommended for
//	             production; interoperates with stock Tor/i2pd/Java-I2P).
//	"auto"     - try native first, fall back to external.
//	"disabled" - do not start this network.
type PrivacyConfig struct {
	Enabled     bool      `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	Persistence bool      `json:"persistence" yaml:"persistence" mapstructure:"persistence"` // persist/reuse identities (default true)
	StateDir    string    `json:"state_dir" yaml:"state_dir" mapstructure:"state_dir"`       // root for persistent identity keys
	Tor         TorConfig `json:"tor" yaml:"tor" mapstructure:"tor"`
	I2P         I2PConfig `json:"i2p" yaml:"i2p" mapstructure:"i2p"`
	Ygg         YggConfig `json:"yggdrasil" yaml:"yggdrasil" mapstructure:"yggdrasil"`
}

type TorConfig struct {
	Mode        string `json:"mode" yaml:"mode" mapstructure:"mode"`                         // native | external | auto | disabled
	DataDir     string `json:"data_dir" yaml:"data_dir" mapstructure:"data_dir"`             // native: persistent tor DataDir
	ControlPort int    `json:"control_port" yaml:"control_port" mapstructure:"control_port"` // native control port (0 = auto)
	SocksPort   int    `json:"socks_port" yaml:"socks_port" mapstructure:"socks_port"`       // external: SOCKS proxy port
	RemotePorts []int  `json:"remote_ports" yaml:"remote_ports" mapstructure:"remote_ports"` // onion virtual ports
	OnionPort   int    `json:"onion_port" yaml:"onion_port" mapstructure:"onion_port"`       // local port the onion forwards to (0 = relay port)
	UseV3       bool   `json:"v3" yaml:"v3" mapstructure:"v3"`
	KeyFile     string `json:"key_file" yaml:"key_file" mapstructure:"key_file"`
}

type I2PConfig struct {
	Mode        string `json:"mode" yaml:"mode" mapstructure:"mode"` // native | external | auto | disabled
	SAMHost     string `json:"sam_host" yaml:"sam_host" mapstructure:"sam_host"`
	SAMPort     int    `json:"sam_port" yaml:"sam_port" mapstructure:"sam_port"`    // external: SAM API port (default 7656)
	I2CPPort    int    `json:"i2cp_port" yaml:"i2cp_port" mapstructure:"i2cp_port"` // native: embedded router I2CP port
	SessionName string `json:"session_name" yaml:"session_name" mapstructure:"session_name"`
	DataDir     string `json:"data_dir" yaml:"data_dir" mapstructure:"data_dir"`
}

type YggConfig struct {
	Mode       string   `json:"mode" yaml:"mode" mapstructure:"mode"` // native | external | auto | disabled
	Peers      []string `json:"peers" yaml:"peers" mapstructure:"peers"`
	DataDir    string   `json:"data_dir" yaml:"data_dir" mapstructure:"data_dir"`
	ListenPort int      `json:"listen_port" yaml:"listen_port" mapstructure:"listen_port"` // Yggdrasil-internal port (0 = relay port)
}

// Enabled reports whether the privacy subsystem is switched on at all.
func (c PrivacyConfig) EnabledNetworks() bool { return c.Enabled }
