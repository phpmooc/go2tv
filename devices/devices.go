package devices

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alexballas/go-ssdp"
	"github.com/pkg/errors"
	"go2tv.app/go2tv/v2/soapcalls"
)

const (
	DeviceTypeDLNA       = "DLNA"
	DeviceTypeChromecast = "Chromecast"
	dlnaDiscoveryDelay   = 2
	dlnaDiscoveryPause   = time.Second
	dlnaLocationTimeout  = 1500 * time.Millisecond
)

type Device struct {
	Name        string
	Addr        string
	Type        string
	IsAudioOnly bool
}

type deviceEntry struct {
	name string
	addr string
}

var ErrNoDeviceAvailable = errors.New("loadSSDPservices: No available Media Renderers")

var (
	ssdpSearch              = ssdp.Search
	loadDevicesFromLocation = soapcalls.LoadDevicesFromLocation
	interfaceAddrs          = net.Interfaces
	ifaceAddrLookup         = func(iface net.Interface) ([]net.Addr, error) { return iface.Addrs() }
	listenUDP               = net.ListenUDP
	discoveryLogOutput      io.Writer
	discoveryLogMu          sync.Mutex
	discoverySummaryState   = make(map[string]summaryState)
	dlnaDevices             []Device
	dlnaMu                  sync.RWMutex
	discoveryStartOnce      sync.Once
)

type summaryState struct {
	lastLine string
	repeats  int
}

// SetDiscoveryLogOutput enables lightweight discovery debug logs.
func SetDiscoveryLogOutput(w io.Writer) {
	discoveryLogMu.Lock()
	discoveryLogOutput = w
	clear(discoverySummaryState)
	discoveryLogMu.Unlock()
}

func discoveryDebugf(format string, args ...any) {
	discoveryLogMu.Lock()
	defer discoveryLogMu.Unlock()

	if discoveryLogOutput == nil {
		return
	}

	_, _ = fmt.Fprintf(discoveryLogOutput, "discovery: "+format+"\n", args...)
}

func discoverySummaryf(key string, format string, args ...any) {
	discoveryLogMu.Lock()
	defer discoveryLogMu.Unlock()

	if discoveryLogOutput == nil {
		return
	}

	line := fmt.Sprintf(format, args...)
	state := discoverySummaryState[key]
	if state.lastLine == line {
		state.repeats++
		discoverySummaryState[key] = state
		return
	}

	if state.repeats > 0 {
		_, _ = fmt.Fprintf(discoveryLogOutput, "discovery: %s repeated=%d\n", key, state.repeats)
	}

	discoverySummaryState[key] = summaryState{lastLine: line}
	_, _ = fmt.Fprintf(discoveryLogOutput, "discovery: %s\n", line)
}

func formatSummaryList(values []string, maxItems int) string {
	if len(values) == 0 {
		return ""
	}

	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	unique := sorted[:0]
	for _, value := range sorted {
		if len(unique) > 0 && unique[len(unique)-1] == value {
			continue
		}
		unique = append(unique, value)
	}

	if maxItems <= 0 || len(unique) <= maxItems {
		return strings.Join(unique, ",")
	}

	return fmt.Sprintf("%s,+%d more", strings.Join(unique[:maxItems], ","), len(unique)-maxItems)
}

func deviceNames(devices []Device, maxItems int) string {
	names := make([]string, 0, len(devices))
	for _, device := range devices {
		names = append(names, device.Name)
	}

	return formatSummaryList(names, maxItems)
}

func locationHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}

	return u.Host
}

// IsChromecastURL returns true if the URL points to a Chromecast device (port 8009).
func IsChromecastURL(deviceURL string) bool {
	u, err := url.Parse(deviceURL)
	if err != nil {
		return false
	}
	return u.Port() == "8009"
}

// LoadSSDPservices returns a slice of DLNA devices that support
// required playback services.
func LoadSSDPservices(delay int) ([]Device, error) {
	// Collect unique locations (a single location may have multiple embedded devices).
	// We intentionally do not filter by ST value here because some vendors reply with
	// non-AVTransport ST values while still exposing AVTransport in LOCATION XML.
	locations := make(map[string]struct{})
	var loadErrors, filteredDevices, unnamedDevices, dupNameCount int
	loadErrorHosts := make([]string, 0)

	port := 3339

	var (
		address *net.UDPAddr
		tries   int
	)

	for address == nil && tries < 100 {
		addr := net.UDPAddr{
			Port: port,
			IP:   net.ParseIP("0.0.0.0"),
		}

		if err := checkInterfacesForPort(port); err != nil {
			port++
			tries++
			continue
		}

		address = &addr
	}

	var addrString string
	if address != nil {
		addrString = address.String()
	}

	list, err := ssdpSearch(ssdp.All, delay, addrString)
	if err != nil {
		discoveryDebugf("LoadSSDPservices search error: %v", err)
		return nil, fmt.Errorf("LoadSSDPservices search error: %w", err)
	}

	for _, srv := range list {
		if srv.Location != "" {
			locations[srv.Location] = struct{}{}
		}
	}

	var allDevices []deviceEntry

	for loc := range locations {
		locCtx, cancel := context.WithTimeout(context.Background(), dlnaLocationTimeout)
		devices, err := loadDevicesFromLocation(locCtx, loc)
		cancel()
		if err != nil {
			loadErrors++
			loadErrorHosts = append(loadErrorHosts, locationHost(loc))
			continue
		}

		for _, dev := range devices {
			if !isDLNADeviceCastable(dev) {
				filteredDevices++
				continue
			}

			name := dev.FriendlyName
			if name == "" {
				name = "Unknown Device"
				unnamedDevices++
			}
			allDevices = append(allDevices, deviceEntry{name: name, addr: loc})
		}
	}

	// Handle duplicate names
	deviceList := make(map[string]string)
	dupNames := make(map[string]int)
	for _, dev := range allDevices {
		fn := dev.name
		_, exists := dupNames[fn]
		dupNames[fn]++
		if exists {
			fn = fn + " (" + dev.addr + ")"
		}

		deviceList[fn] = dev.addr
	}

	for fn, c := range dupNames {
		if c > 1 {
			dupNameCount++
			loc := deviceList[fn]
			delete(deviceList, fn)
			fn = fn + " (" + loc + ")"
			deviceList[fn] = loc
		}
	}

	if len(deviceList) == 0 {
		discoverySummaryf(
			"LoadSSDPservices summary",
			"LoadSSDPservices summary listen_addr=%q ssdp_results=%d unique_locations=%d load_errors=%d error_hosts=%q filtered=%d duplicate_names=%d unnamed=%d returned=0 devices=%q",
			addrString,
			len(list),
			len(locations),
			loadErrors,
			formatSummaryList(loadErrorHosts, 3),
			filteredDevices,
			dupNameCount,
			unnamedDevices,
			"",
		)
		return nil, ErrNoDeviceAvailable
	}

	// Convert map to Device slice with proper type
	result := make([]Device, 0, len(deviceList))
	for name, addr := range deviceList {
		result = append(result, Device{
			Name: name,
			Addr: addr,
			Type: DeviceTypeDLNA,
		})
	}
	discoverySummaryf(
		"LoadSSDPservices summary",
		"LoadSSDPservices summary listen_addr=%q ssdp_results=%d unique_locations=%d load_errors=%d error_hosts=%q filtered=%d duplicate_names=%d unnamed=%d returned=%d devices=%q",
		addrString,
		len(list),
		len(locations),
		loadErrors,
		formatSummaryList(loadErrorHosts, 3),
		filteredDevices,
		dupNameCount,
		unnamedDevices,
		len(result),
		deviceNames(result, 3),
	)

	return result, nil
}

// StartDiscovery starts the background discovery loops once for the process.
func StartDiscovery(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	discoveryStartOnce.Do(func() {
		discoveryDebugf("Discovery loops starting")
		StartChromecastDiscoveryLoop(ctx)
		startDLNADiscoveryLoop(ctx)
	})
}

func startDLNADiscoveryLoop(ctx context.Context) {
	go func() {
		pollTimer := time.NewTimer(0)
		defer pollTimer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTimer.C:
			}

			refreshDLNADevices()
			pollTimer.Reset(dlnaDiscoveryPause)
		}
	}()
}

func refreshDLNADevices() {
	devices, err := LoadSSDPservices(dlnaDiscoveryDelay)
	if err != nil {
		if stderrors.Is(err, ErrNoDeviceAvailable) {
			setDLNADevices(nil)
			return
		}

		discoveryDebugf("DLNA discovery scan error: %v", err)
		return
	}

	setDLNADevices(devices)
}

func setDLNADevices(devices []Device) {
	dlnaMu.Lock()
	dlnaDevices = append([]Device(nil), devices...)
	dlnaMu.Unlock()
}

func getDLNADevices() []Device {
	dlnaMu.RLock()
	defer dlnaMu.RUnlock()

	return append([]Device(nil), dlnaDevices...)
}
func isDLNADeviceCastable(dev *soapcalls.DMRextracted) bool {
	if dev == nil {
		return false
	}

	return dev.AvtransportControlURL != ""
}

// LoadAllDevices returns the latest cached DLNA and Chromecast devices.
func LoadAllDevices() ([]Device, error) {
	dlna := getDLNADevices()
	chromecast := getChromecastDevicesSnapshot()

	combined := make([]Device, 0, len(dlna)+len(chromecast))
	combined = append(combined, dlna...)
	combined = append(combined, chromecast...)

	if len(combined) == 0 {
		discoverySummaryf(
			"LoadAllDevices summary",
			"LoadAllDevices summary cached=true dlna=%d dlna_names=%q chromecast=%d chromecast_names=%q combined=0",
			len(dlna),
			deviceNames(dlna, 3),
			len(chromecast),
			deviceNames(chromecast, 3),
		)
		return nil, ErrNoDeviceAvailable
	}

	sortDevices(combined)
	discoverySummaryf(
		"LoadAllDevices summary",
		"LoadAllDevices summary cached=true dlna=%d dlna_names=%q chromecast=%d chromecast_names=%q combined=%d combined_names=%q",
		len(dlna),
		deviceNames(dlna, 3),
		len(chromecast),
		deviceNames(chromecast, 3),
		len(combined),
		deviceNames(combined, 4),
	)

	return combined, nil
}

// sortDevices sorts devices by type (DLNA first, then Chromecast)
// and alphabetically within each type
func sortDevices(devices []Device) {
	sort.Slice(devices, func(i, j int) bool {
		// If types differ, DLNA comes first
		if devices[i].Type != devices[j].Type {
			return devices[i].Type < devices[j].Type
		}
		// Within same type, sort alphabetically by name
		return devices[i].Name < devices[j].Name
	})
}

func checkInterfacesForPort(port int) error {
	interfaces, err := interfaceAddrs()
	if err != nil {
		return err
	}

	var lastErr error

	for _, iface := range interfaces {
		addrs, err := ifaceAddrLookup(iface)
		if err != nil {
			lastErr = err
			continue
		}

		for _, addr := range addrs {
			ip, _, _ := net.ParseCIDR(addr.String())

			if ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			udpAddr := net.UDPAddr{
				Port: port,
				IP:   ip,
			}

			udpConn, err := listenUDP("udp4", &udpAddr)
			if err != nil {
				lastErr = err
				continue
			}

			udpConn.Close()
			return nil

		}
	}

	if lastErr != nil {
		return lastErr
	}

	return nil
}
