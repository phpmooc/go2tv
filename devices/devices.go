package devices

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/alexballas/go-ssdp"
	"github.com/pkg/errors"
	"go2tv.app/go2tv/v2/soapcalls"
)

const (
	DeviceTypeDLNA       = "DLNA"
	DeviceTypeChromecast = "Chromecast"
)

type Device struct {
	Name        string
	Addr        string
	Type        string
	IsAudioOnly bool
}

var (
	ErrNoDeviceAvailable  = errors.New("loadSSDPservices: No available Media Renderers")
	ErrDeviceNotAvailable = errors.New("devicePicker: Requested device not available")
	ErrSomethingWentWrong = errors.New("devicePicker: Something went terribly wrong")
)

var (
	ssdpSearch              = ssdp.Search
	loadDevicesFromLocation = soapcalls.LoadDevicesFromLocation
	discoveryLogOutput      io.Writer
	discoveryLogMu          sync.Mutex
)

// SetDiscoveryLogOutput enables lightweight discovery debug logs.
func SetDiscoveryLogOutput(w io.Writer) {
	discoveryLogMu.Lock()
	discoveryLogOutput = w
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
	discoveryDebugf("LoadSSDPservices start delay=%d", delay)

	// Collect unique locations (a single location may have multiple embedded devices).
	// We intentionally do not filter by ST value here because some vendors reply with
	// non-AVTransport ST values while still exposing AVTransport in LOCATION XML.
	locations := make(map[string]struct{})

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
	discoveryDebugf("LoadSSDPservices ssdp_results=%d listen_addr=%q", len(list), addrString)

	for _, srv := range list {
		if srv.Location != "" {
			discoveryDebugf("LoadSSDPservices ssdp_hit type=%q location=%q", srv.Type, srv.Location)
			locations[srv.Location] = struct{}{}
		}
	}

	// Extract all devices from each unique location
	type deviceEntry struct {
		name string
		addr string
	}
	var allDevices []deviceEntry

	for loc := range locations {
		devices, err := loadDevicesFromLocation(context.Background(), loc)
		if err != nil {
			discoveryDebugf("LoadSSDPservices location=%q load error: %v", loc, err)
			continue
		}
		discoveryDebugf("LoadSSDPservices location=%q parsed_devices=%d", loc, len(devices))

		for _, dev := range devices {
			if !isDLNADeviceCastable(dev) {
				discoveryDebugf(
					"LoadSSDPservices filtered name=%q udn=%q avtransport=%t connection_manager=%t",
					dev.FriendlyName,
					dev.UDN,
					dev.AvtransportControlURL != "",
					dev.ConnectionManagerURL != "",
				)
				continue
			}

			name := dev.FriendlyName
			if name == "" {
				name = "Unknown Device"
				discoveryDebugf("LoadSSDPservices empty friendly name location=%q", loc)
			}
			discoveryDebugf("LoadSSDPservices accepted name=%q location=%q", name, loc)
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
			loc := deviceList[fn]
			delete(deviceList, fn)
			fn = fn + " (" + loc + ")"
			deviceList[fn] = loc
		}
	}

	if len(deviceList) == 0 {
		discoveryDebugf("LoadSSDPservices no devices after filtering")
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
	discoveryDebugf("LoadSSDPservices returning_devices=%d", len(result))

	return result, nil
}

func isDLNADeviceCastable(dev *soapcalls.DMRextracted) bool {
	if dev == nil {
		return false
	}

	// AVTransport drives playback. ConnectionManager is required by Play1 path
	// because we call GetProtocolInfo before SetAVTransportURI.
	return dev.AvtransportControlURL != "" && dev.ConnectionManagerURL != ""
}

// LoadAllDevices returns a combined slice of DLNA and Chromecast devices.
// It runs both discovery mechanisms concurrently and returns partial results
// immediately without waiting for both to complete. This ensures delays in
// one protocol don't block the other.
func LoadAllDevices(delay int) ([]Device, error) {
	discoveryDebugf("LoadAllDevices start delay=%d", delay)

	type deviceResult struct {
		devices []Device
		err     error
	}

	dlnaChan := make(chan deviceResult, 1)
	ccChan := make(chan deviceResult, 1)

	// Launch DLNA discovery in background
	go func() {
		devices, err := LoadSSDPservices(delay)
		dlnaChan <- deviceResult{devices: devices, err: err}
	}()

	// Launch Chromecast discovery in background (instant, reads from cache)
	go func() {
		devices := GetChromecastDevices()
		ccChan <- deviceResult{devices: devices, err: nil}
	}()

	// Collect results as they arrive, with timeout
	combined := make([]Device, 0)
	timeout := time.After(time.Duration(delay+1) * time.Second)
	resultsReceived := 0

	for resultsReceived < 2 {
		select {
		case result := <-dlnaChan:
			discoveryDebugf("LoadAllDevices dlna_result devices=%d err=%v", len(result.devices), result.err)
			if result.err == nil {
				combined = append(combined, result.devices...)
			}
			resultsReceived++

		case result := <-ccChan:
			discoveryDebugf("LoadAllDevices chromecast_result devices=%d", len(result.devices))
			if result.devices != nil {
				combined = append(combined, result.devices...)
			}
			resultsReceived++

		case <-timeout:
			discoveryDebugf("LoadAllDevices timeout combined=%d results_received=%d", len(combined), resultsReceived)
			// Return partial results if timeout occurs
			if len(combined) > 0 {
				sortDevices(combined)
				discoveryDebugf("LoadAllDevices returning partial devices=%d", len(combined))
				return combined, nil
			}
			discoveryDebugf("LoadAllDevices no devices before timeout")
			return nil, ErrNoDeviceAvailable
		}
	}

	if len(combined) > 0 {
		sortDevices(combined)
		discoveryDebugf("LoadAllDevices returning_devices=%d", len(combined))
		return combined, nil
	}

	discoveryDebugf("LoadAllDevices no devices returned")
	return nil, ErrNoDeviceAvailable
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

// DevicePicker will pick the nth device from the devices input slice.
func DevicePicker(devices []Device, n int) (string, error) {
	if n > len(devices) || len(devices) == 0 || n <= 0 {
		return "", ErrDeviceNotAvailable
	}

	if n > len(devices) {
		return "", ErrDeviceNotAvailable
	}

	return devices[n-1].Addr, nil
}

func checkInterfacesForPort(port int) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return err
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

			udpConn, err := net.ListenUDP("udp4", &udpAddr)
			if err != nil {
				return err
			}

			udpConn.Close()
			return nil

		}
	}

	return nil
}
