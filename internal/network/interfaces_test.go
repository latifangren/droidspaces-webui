package network

import (
	"testing"
)

func TestClassifyInterfaceType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"wlan0", "wifi"},
		{"wifi_direct", "wifi"},
		{"rmnet0", "cellular"},
		{"ccmni1", "cellular"},
		{"pdp_data", "cellular"},
		{"wwan0", "cellular"},
		{"eth0", "ethernet"},
		{"br-lan", "bridge"},
		{"ds-net", "bridge"},
		{"docker0", "bridge"},
		{"rndis0", "usb_tether"},
		{"usb0", "usb_tether"},
		{"custom_iface", "other"},
	}

	for _, tc := range tests {
		got := ClassifyInterfaceType(tc.name)
		if got != tc.expected {
			t.Errorf("for interface %q: expected %q, got %q", tc.name, tc.expected, got)
		}
	}
}

func TestListHostNetworkInterfaces(t *testing.T) {
	ifaces, err := ListHostNetworkInterfaces()
	if err != nil {
		t.Fatalf("ListHostNetworkInterfaces failed: %v", err)
	}

	// Should not panic or crash, and if interfaces exist, verify sorting and no 'lo'
	for i, ifi := range ifaces {
		if ifi.Name == "lo" {
			t.Errorf("loopback interface 'lo' should be excluded")
		}
		if i > 0 && ifaces[i-1].Name > ifi.Name {
			t.Errorf("interfaces should be sorted by name: %s > %s", ifaces[i-1].Name, ifi.Name)
		}
		if ifi.State != "up" && ifi.State != "down" {
			t.Errorf("expected state 'up' or 'down', got %q", ifi.State)
		}
	}
}
