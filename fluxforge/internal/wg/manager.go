//go:build linux

package wg

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// PeerConfig describes a single WireGuard peer to configure.
type PeerConfig struct {
	PublicKey string // WireGuard public key, base64
	AllowedIP string // CIDR e.g. "10.99.2.1/32"
	Endpoint  string // host:port, empty if unknown
}

// Manager owns a WireGuard network interface for the duration of the process.
// It tries kernel WireGuard first; falls back to wireguard-go userspace.
type Manager struct {
	mu       sync.Mutex
	iface    string
	client   *wgctrl.Client
	dev      *device.Device // non-nil when using wireguard-go userspace
	uapiStop func()         // stops the UAPI listener goroutine
}

// New creates the WireGuard interface, assigns meshIP, and configures privKey.
// meshIP must be bare e.g. "10.99.1.1" — the /16 mask is added automatically.
func New(iface, meshIP string, privKey wgtypes.Key, listenPort int) (*Manager, error) {
	m := &Manager{iface: iface}

	// Try kernel WireGuard (requires wireguard kernel module).
	if err := m.createKernelInterface(); err != nil {
		// Fall back to wireguard-go userspace.
		if err := m.createUserspaceInterface(); err != nil {
			return nil, fmt.Errorf("creating WireGuard interface: %w", err)
		}
	}

	if err := assignIP(iface, meshIP); err != nil {
		m.cleanup()
		return nil, fmt.Errorf("assigning mesh IP: %w", err)
	}

	client, err := wgctrl.New()
	if err != nil {
		m.cleanup()
		return nil, fmt.Errorf("wgctrl: %w", err)
	}
	m.client = client

	port := listenPort
	if err := client.ConfigureDevice(iface, wgtypes.Config{
		PrivateKey: &privKey,
		ListenPort: &port,
	}); err != nil {
		m.cleanup()
		return nil, fmt.Errorf("configuring WireGuard keys: %w", err)
	}

	return m, nil
}

func (m *Manager) createKernelInterface() error {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = m.iface
	// GenericLink with type "wireguard" creates a kernel WireGuard device.
	link := &netlink.GenericLink{
		LinkAttrs: attrs,
		LinkType:  "wireguard",
	}
	return netlink.LinkAdd(link)
}

func (m *Manager) createUserspaceInterface() error {
	tdev, err := tun.CreateTUN(m.iface, device.DefaultMTU)
	if err != nil {
		return fmt.Errorf("creating TUN: %w", err)
	}

	logger := device.NewLogger(device.LogLevelSilent, "")
	dev := device.NewDevice(tdev, conn.NewDefaultBind(), logger)

	// Open the UAPI socket so wgctrl can configure this device.
	uapiFile, err := ipc.UAPIOpen(m.iface)
	if err != nil {
		dev.Close()
		return fmt.Errorf("UAPI open: %w", err)
	}

	uapiLn, err := ipc.UAPIListen(m.iface, uapiFile)
	if err != nil {
		dev.Close()
		uapiFile.Close()
		return fmt.Errorf("UAPI listen: %w", err)
	}

	stop := make(chan struct{})
	go func() {
		for {
			c, err := uapiLn.Accept()
			if err != nil {
				select {
				case <-stop:
				default:
					// Listener closed externally.
				}
				return
			}
			go dev.IpcHandle(c)
		}
	}()

	m.dev = dev
	m.uapiStop = func() {
		close(stop)
		uapiLn.Close()
		uapiFile.Close()
	}
	return nil
}

func assignIP(iface, meshIP string) error {
	ip := net.ParseIP(meshIP)
	if ip == nil {
		return fmt.Errorf("invalid mesh IP: %s", meshIP)
	}

	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("interface not found: %w", err)
	}

	addr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip.To4(),
			Mask: net.CIDRMask(16, 32),
		},
	}
	if err := netlink.AddrAdd(link, addr); err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("adding IP: %w", err)
	}

	return netlink.LinkSetUp(link)
}

// SetPeers atomically replaces all WireGuard peers.
func (m *Manager) SetPeers(peers []PeerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wgPeers := make([]wgtypes.PeerConfig, 0, len(peers))
	for _, p := range peers {
		pub, err := wgtypes.ParseKey(p.PublicKey)
		if err != nil {
			return fmt.Errorf("parsing public key: %w", err)
		}

		var ep *net.UDPAddr
		if p.Endpoint != "" {
			ep, err = net.ResolveUDPAddr("udp", p.Endpoint)
			if err != nil {
				return fmt.Errorf("resolving endpoint %s: %w", p.Endpoint, err)
			}
		}

		_, cidr, err := net.ParseCIDR(p.AllowedIP)
		if err != nil {
			return fmt.Errorf("parsing allowed IP %s: %w", p.AllowedIP, err)
		}

		keepalive := 25 * time.Second
		wgPeers = append(wgPeers, wgtypes.PeerConfig{
			PublicKey:                   pub,
			Endpoint:                    ep,
			AllowedIPs:                  []net.IPNet{*cidr},
			PersistentKeepaliveInterval: &keepalive,
		})
	}

	return m.client.ConfigureDevice(m.iface, wgtypes.Config{
		ReplacePeers: true,
		Peers:        wgPeers,
	})
}

// Close tears down the WireGuard interface and releases all resources.
func (m *Manager) Close() error {
	if m.uapiStop != nil {
		m.uapiStop()
	}
	if m.client != nil {
		m.client.Close()
	}
	if m.dev != nil {
		m.dev.Close()
	}
	m.removeInterface()
	return nil
}

func (m *Manager) cleanup() {
	if m.uapiStop != nil {
		m.uapiStop()
	}
	if m.client != nil {
		m.client.Close()
	}
	if m.dev != nil {
		m.dev.Close()
	}
}

func (m *Manager) removeInterface() {
	link, err := netlink.LinkByName(m.iface)
	if err == nil {
		netlink.LinkDel(link)
	}
}

// GenerateKey generates a new WireGuard private key.
func GenerateKey() (wgtypes.Key, error) {
	return wgtypes.GeneratePrivateKey()
}

// PublicKey returns the public key corresponding to a private key.
func PublicKey(priv wgtypes.Key) wgtypes.Key {
	return priv.PublicKey()
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return err == os.ErrExist || err.Error() == "file exists"
}
