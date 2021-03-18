package cmd

import (
	"time"

	"github.com/combust-labs/firebox/config"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/spf13/cobra"
)

var (
	vmmConfig = new(config.VMMConfig)
)

func initVMMConfigFlags(cmd *cobra.Command) {
	cmd.Flags().DurationVar(&vmmConfig.VMM.ShutdownTimeout, "shutdown-timeout", 30*time.Second, "Shutdown timeout before VMM is stopped forcefully")

	cmd.Flags().StringVar(&vmmConfig.RootFS, "rootfs", "./image.ext4", "Path to root disk image")
	cmd.Flags().StringVar(&vmmConfig.KernelImage, "kernel-image", "./vmlinux", "Path to the kernel image")
	cmd.Flags().StringVar(&vmmConfig.KernelArgs, "kernel-args", "console=ttyS0 noapic reboot=k panic=1 pci=off nomodules rw", "The command-line arguments that should be passed to the kernel.")
	cmd.Flags().StringVar(&vmmConfig.NetNS, "net-ns", "", "Network namespace")

	cmd.Flags().StringVar(&vmmConfig.SocketPath, "socket-path", "", "Path to use for firecracker socket, defaults to a unique file in in the first existing directory from {$HOME, $TMPDIR, or /tmp}")
	cmd.Flags().StringVar(&vmmConfig.LogLevel, "machine-log-level", models.LoggerLevelDebug, "Verbosity of Firecracker logging.  One of: Debug, Info, Warning or Error")
	cmd.Flags().BoolVar(&vmmConfig.DebugClient, "machine-debug-client", false, "Debug firecracker HTTP calls. Requires machine log level debug.")

	cmd.Flags().BoolVar(&vmmConfig.Network.AllowMMDS, "mmds", false, "Activate the microVM Metadata Service")

	cmd.Flags().StringVar(&vmmConfig.Network.CNI.BinDir, "cni-bin-dir", "/opt/cni/bin", "CNI plugins binaries directory")
	cmd.Flags().StringVar(&vmmConfig.Network.CNI.ConfDir, "cni-conf-dir", "/etc/cni/conf.d", "CNI configuration directory")
	cmd.Flags().StringVar(&vmmConfig.Network.CNI.CacheDir, "cni-cache-dir", "/var/lib/cni", "CNI cache directory")

	cmd.Flags().BoolVar(&vmmConfig.Network.CNI.Enable, "cni-enable", true, "Enable CNI")
	cmd.Flags().StringVar(&vmmConfig.Network.CNI.NetworkName, "cni-network-name", "firebox", "Name in the Network Configuration List")
	cmd.Flags().StringVar(&vmmConfig.Network.CNI.IfaceName, "cni-iface-name", "", "Network interface name")

	cmd.Flags().BoolVar(&vmmConfig.Jailer.Enable, "jailer-enable", false, "Enable jailer usage")
	cmd.Flags().StringVar(&vmmConfig.Jailer.ExecFile, "jailer-exec-file", "/usr/local/bin/firecracker", "Path to firecracker binary")
	cmd.Flags().StringVar(&vmmConfig.Jailer.JailerBinary, "jailer-binary", "/usr/local/bin/jailer", "Path to jailer binary")
	cmd.Flags().StringVar(&vmmConfig.Jailer.VMID, "jailer-vm-id", "", "ID is the unique VM identification string, which may contain alphanumeric characters and hyphens. The maximum id length is currently 64 characters")
	cmd.Flags().IntVar(&vmmConfig.Jailer.UID, "jailer-uid", 0, "Jailer uid for dropping privileges")
	cmd.Flags().IntVar(&vmmConfig.Jailer.GID, "jailer-gid", 0, "Jailer gid for dropping privileges")
	cmd.Flags().IntVar(&vmmConfig.Jailer.NumaNode, "jailer-numa-node", 0, "Jailer numa node")

	cmd.Flags().StringVar(&vmmConfig.Jailer.ChrootBaseDir, "jailer-chroot-base-dir", "/srv/jailer", "Jailer chroot base directory")
	cmd.Flags().BoolVar(&vmmConfig.Jailer.Daemonize, "jailer-daemonize", false, "if set to true, create a new session (setsid) and redirect STDIN, STDOUT, and STDERR to /dev/null")

	cmd.Flags().StringVar(&vmmConfig.Machine.CPUTemplate, "machine-cpu-template", "", "CPU template (T2/ C3)")
	cmd.Flags().BoolVar(&vmmConfig.Machine.HtEnabled, "machine-ht-enabled", false, "Enable hyperthreading")
	cmd.Flags().Int64Var(&vmmConfig.Machine.MemSizeMib, "machine-mem-size", 128, "Memory size of VM in Mib")
	cmd.Flags().Int64Var(&vmmConfig.Machine.VcpuCount, "machine-vcpu_count", 1, "Number of vCPUs (either 1 or an even number)")
}
