package config

import "time"

type JailerConfig struct {
	Enable        bool
	ExecFile      string
	JailerBinary  string
	VMID          string
	GID           int
	UID           int
	NumaNode      int
	ChrootBaseDir string
	Daemonize     bool
}

type CNIConfig struct {
	BinDir      string
	ConfDir     string
	CacheDir    string
	Enable      bool
	NetworkName string
	IfaceName   string
}

type VMMConfig struct {
	SocketPath  string
	LogLevel    string
	DebugClient bool
	RootFS      string
	KernelImage string
	KernelArgs  string
	NetNS       string
	Jailer      JailerConfig
	Machine     struct {
		CPUTemplate string
		HtEnabled   bool
		MemSizeMib  int64
		VcpuCount   int64
	}
	Network struct {
		CNI CNIConfig
	}
	VMM struct {
		ShutdownTimeout time.Duration
	}
}
