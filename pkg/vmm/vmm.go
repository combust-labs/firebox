package vmm

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/combust-labs/firebox/config"
	"github.com/combust-labs/firebox/pkg/log"
	"github.com/containernetworking/cni/libcni"
	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// Avoid "path must be shorter than SUN_LEN" error
const MaxSocketPathLength = 107

type VMM interface {
	Start() error
	WaitFinished() error
	Stop() error
	GetIP() net.IP
	GetID() string
}

type vmm struct {
	logger *log.Logger

	fcConfig        *firecracker.Config
	vmmCtx          context.Context
	shutdownTimeout time.Duration
	vmmConfig       config.VMMConfig

	machine *firecracker.Machine
}

func NewVMM(logger *log.Logger, vmmConfig config.VMMConfig) VMM {
	var fifo io.WriteCloser
	vmmID := uuid.Must(uuid.NewV4()).String()
	logger.Infof("Starting VMM ID %s", vmmID)

	fcConfig := &firecracker.Config{
		SocketPath:        getSocketPath(&vmmConfig),
		LogFifo:           "",
		LogLevel:          vmmConfig.LogLevel,
		MetricsFifo:       "",
		KernelImagePath:   vmmConfig.KernelImage,
		InitrdPath:        "",
		KernelArgs:        vmmConfig.KernelArgs,
		Drives:            firecracker.NewDrivesBuilder(vmmConfig.RootFS).Build(),
		NetworkInterfaces: getNetworkInterfaces(&vmmConfig),
		FifoLogWriter:     fifo,
		VsockDevices:      []firecracker.VsockDevice{},
		MachineCfg: models.MachineConfiguration{
			CPUTemplate: models.CPUTemplate(vmmConfig.Machine.CPUTemplate),
			HtEnabled:   firecracker.Bool(vmmConfig.Machine.HtEnabled),
			MemSizeMib:  firecracker.Int64(vmmConfig.Machine.MemSizeMib),
			VcpuCount:   firecracker.Int64(vmmConfig.Machine.VcpuCount),
		},
		DisableValidation: false,
		JailerCfg:         getJailerConfig(&vmmConfig),
		VMID:              vmmID,
		NetNS:             vmmConfig.NetNS,
		ForwardSignals:    nil,
		SeccompLevel:      firecracker.SeccompLevelDisable,
	}
	return &vmm{
		logger:          logger,
		fcConfig:        fcConfig,
		vmmCtx:          context.Background(),
		shutdownTimeout: vmmConfig.VMM.ShutdownTimeout,
		vmmConfig:       vmmConfig,
	}
}

func (f *vmm) GetID() string {
	return f.fcConfig.VMID
}

func (f *vmm) Start() error {
	machine, err := f.runVMM(f.vmmCtx)
	if err != nil {
		return errors.Wrap(err, "runVMM failed")
	}
	f.machine = machine
	return nil
}

func (f *vmm) WaitFinished() error {
	if err := f.machine.Wait(f.vmmCtx); err != nil {
		return errors.Wrap(err, "machine.Wait failed")
	}
	return nil
}

func (f *vmm) Stop() error {
	if f.machine != nil {
		f.stopVMM(f.vmmCtx, f.machine)
	}
	return nil
}

func (f *vmm) GetIP() net.IP {
	if len(f.fcConfig.NetworkInterfaces) > 0 {
		ni0 := &f.fcConfig.NetworkInterfaces[0]
		if ni0.StaticConfiguration != nil && ni0.StaticConfiguration.IPConfiguration != nil {
			return ni0.StaticConfiguration.IPConfiguration.IPAddr.IP
		}
	}
	return nil
}

func (f *vmm) runVMM(ctx context.Context) (*firecracker.Machine, error) {
	logger := f.logger.RawLogger().WithField("vmid", f.fcConfig.VMID).WithField("subsystem", "firecracker-sdk")

	opts := []firecracker.Opt{
		firecracker.WithLogger(logger),
	}
	m, err := firecracker.NewMachine(ctx, *f.fcConfig, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Machine creation failed")
	}
	if f.vmmConfig.DebugClient {
		opts = append(opts, firecracker.WithClient(firecracker.NewClient(m.Cfg.SocketPath, logger, true)))
		m, err = firecracker.NewMachine(ctx, *f.fcConfig, opts...)
		if err != nil {
			return nil, errors.Wrap(err, "Machine creation with debug failed")
		}
	}
	if len(m.Cfg.SocketPath) > MaxSocketPathLength {
		return nil, errors.Errorf("Socket path too long %d, this will generate 'path must be shorter than SUN_LEN'", len(m.Cfg.SocketPath))
	}

	if err := m.Start(ctx); err != nil {
		return nil, errors.Wrap(err, "Machine start failed")
	}
	return m, nil
}

func (f *vmm) stopVMM(ctx context.Context, machine *firecracker.Machine) {
	f.logger.Infof("stopping VMM ID %v", machine.Cfg.VMID)

	shutdownCtx, cancelFunc := context.WithTimeout(ctx, f.shutdownTimeout)
	defer cancelFunc()

	chanStopped := make(chan error, 1)
	go func() {
		chanStopped <- machine.Shutdown(shutdownCtx)
	}()

	select {
	case stopErr := <-chanStopped:
		if stopErr != nil {
			f.logger.Warnf("VMM stopped with error but within timeout: %v", stopErr)
			f.logger.Warnf("VMM stopped forcefully: %v", machine.StopVMM()) // force stop
		} else {
			f.logger.Warn("VMM stopped gracefully")
		}
	case <-shutdownCtx.Done():
		f.logger.Warnf("VMM failed to stop gracefully: timeout reached")
		f.logger.Warnf("VMM stopped forcefully: %v ", machine.StopVMM()) // force stop
	}

	for _, iface := range f.fcConfig.NetworkInterfaces {
		cni := iface.CNIConfiguration
		if cni != nil {
			if err := f.cleanupCNINetwork(machine.Cfg.VMID, f.fcConfig.NetNS, cni.NetworkName, cni.IfName, &f.vmmConfig.Network.CNI); err != nil {
				f.logger.Errorf("CNI cleanup failed: %v", err)
			}
		}
	}
	if f.fcConfig.JailerCfg != nil {
		if err := f.cleanupJailerChrootBaseDir(machine.Cfg.JailerCfg.ID, &f.vmmConfig.Jailer); err != nil {
			f.logger.Errorf("chroot dir cleanup failed: %v", err)
		}
	}
}

func (f *vmm) cleanupCNINetwork(vmmID, netNS, networkName, ifname string, cni *config.CNIConfig) error {
	f.logger.Infof("cleaning up CNI network '%v' , ifname '%v' and netns '%v'", networkName, ifname, netNS)

	cniPlugin := libcni.NewCNIConfigWithCacheDir([]string{cni.BinDir}, cni.CacheDir, nil)
	networkConfig, err := libcni.LoadConfList(cni.ConfDir, networkName)
	if err != nil {
		return errors.Wrap(err, "LoadConfList failed")
	}
	if err := cniPlugin.DelNetworkList(context.Background(), networkConfig, &libcni.RuntimeConf{
		ContainerID: vmmID, // golang firecracker SDK uses the VMID, if VMID is set
		NetNS:       netNS,
		IfName:      ifname,
	}); err != nil {
		return errors.Wrap(err, "DelNetworkList failed")
	}
	// clean up the CNI interface directory:
	ifaceCNIDir := filepath.Join(cni.CacheDir, vmmID)

	f.logger.Infof("cleaning up CNI interface directory '%v'", ifaceCNIDir)

	ifaceCNIDirStat, err := os.Stat(ifaceCNIDir)
	if err != nil {
		return errors.Wrapf(err, "os.Stat(%s) failed", ifaceCNIDir)
	}
	if !ifaceCNIDirStat.IsDir() {
		return errors.Wrapf(err, "%s is not a dir", ifaceCNIDir)
	}
	err = os.RemoveAll(ifaceCNIDir)
	if err != nil {
		return errors.Wrapf(err, "RemoveAll from %s failed", ifaceCNIDir)
	}
	return nil
}

func (f *vmm) cleanupJailerChrootBaseDir(jailerVMId string, jailer *config.JailerConfig) error {
	// delete only from /srv subdir - prevent OS deletion  ;-)
	if !strings.HasPrefix(jailer.ChrootBaseDir, "/srv/") {
		return errors.Errorf("Recursive chroot base dir deletion only from /srv subdir but used %v", jailer.ChrootBaseDir)
	}
	execFileName := path.Base(jailer.ExecFile)
	chrootDir := path.Join(jailer.ChrootBaseDir, execFileName, jailerVMId)
	f.logger.Infof("Remove change root dir %v", chrootDir)
	err := os.RemoveAll(chrootDir)
	if err != nil {
		return errors.Wrapf(err, "RemoveAll from %v failed", chrootDir)
	}
	return nil
}

func getJailerConfig(c *config.VMMConfig) *firecracker.JailerConfig {
	if !c.Jailer.Enable {
		return nil
	}
	id := c.Jailer.VMID
	if id == "" {
		id = uuid.Must(uuid.NewV4()).String()
	}
	return &firecracker.JailerConfig{
		ID:             id,
		GID:            firecracker.Int(c.Jailer.GID),
		UID:            firecracker.Int(c.Jailer.UID),
		NumaNode:       firecracker.Int(c.Jailer.NumaNode),
		ExecFile:       c.Jailer.ExecFile,
		JailerBinary:   c.Jailer.JailerBinary,
		ChrootBaseDir:  c.Jailer.ChrootBaseDir,
		Daemonize:      c.Jailer.Daemonize,
		ChrootStrategy: firecracker.NewNaiveChrootStrategy(c.KernelImage),
		Stdout:         os.Stdout,
		Stderr:         os.Stderr,
		Stdin:          nil,
	}
}

func getNetworkInterfaces(c *config.VMMConfig) firecracker.NetworkInterfaces {
	vethIfaceName := c.Network.CNI.IfaceName
	if vethIfaceName == "" {
		vethIfaceName = getRandomVethName()
	}
	return firecracker.NetworkInterfaces{
		firecracker.NetworkInterface{
			// TODO: check if cni is enabled
			CNIConfiguration: &firecracker.CNIConfiguration{
				// NetworkConfig: TODO: provide fcConfig
				NetworkName: c.Network.CNI.NetworkName,
				IfName:      vethIfaceName,
			},
			AllowMMDS: c.Network.AllowMMDS,
		},
	}
}

func getSocketPath(c *config.VMMConfig) string {
	if c.Jailer.Enable {
		// given via Jailer
		return ""
	}
	if c.SocketPath != "" {
		return c.SocketPath
	}
	filename := strings.Join([]string{
		".firecracker.sock",
		strconv.Itoa(os.Getpid()),
		strconv.Itoa(rand.Intn(1000))},
		"-",
	)
	var dir string
	if d := os.Getenv("HOME"); checkExistsAndDir(d) {
		dir = d
	} else if checkExistsAndDir(os.TempDir()) {
		dir = os.TempDir()
	} else {
		dir = "./"
	}
	return filepath.Join(dir, filename)
}

func checkExistsAndDir(path string) bool {
	if path == "" {
		return false
	}
	if info, err := os.Stat(path); err == nil {
		return info.IsDir()
	}
	return false
}

func getRandomVethName() string {
	const n = 11
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("veth%s", string(b))
}
