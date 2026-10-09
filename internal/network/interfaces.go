package network

import (
	"net"
	"sort"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/model"
)

// ClassifyInterfaceType determines the device class from interface name.
func ClassifyInterfaceType(name string) string {
	switch {
	case strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "wifi"):
		return "wifi"
	case strings.HasPrefix(name, "rmnet") || strings.HasPrefix(name, "ccmni") || strings.HasPrefix(name, "pdp") || strings.HasPrefix(name, "wwan"):
		return "cellular"
	case strings.HasPrefix(name, "eth"):
		return "ethernet"
	case strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "ds-") || strings.HasPrefix(name, "docker"):
		return "bridge"
	case strings.HasPrefix(name, "rndis") || strings.HasPrefix(name, "usb"):
		return "usb_tether"
	default:
		return "other"
	}
}

// ListHostNetworkInterfaces discovers host network interfaces and categorizes their types.
func ListHostNetworkInterfaces() ([]model.NetworkInterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []model.NetworkInterfaceInfo
	for _, ifi := range ifaces {
		name := ifi.Name
		if name == "lo" {
			continue
		}

		state := "down"
		if ifi.Flags&net.FlagUp != 0 {
			state = "up"
		}

		ifType := ClassifyInterfaceType(name)

		var ip string
		if addrs, err := ifi.Addrs(); err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						ip = ipnet.IP.String()
						break
					}
				}
			}
		}

		result = append(result, model.NetworkInterfaceInfo{
			Name:  name,
			IP:    ip,
			Type:  ifType,
			State: state,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}
