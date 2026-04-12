package uAddress

import (
	"testing"
)

func TestIsIntranet(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"10段", "10.0.0.1", true},
		{"172段", "172.16.0.1", true},
		{"172段边界", "172.31.255.255", true},
		{"172段外", "172.32.0.1", false},
		{"192段", "192.168.1.1", true},
		{"20段", "20.0.0.1", true},
		{"公网", "8.8.8.8", false},
		{"公网2", "114.114.114.114", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIntranet(tt.ip)
			if got != tt.expected {
				t.Errorf("IsIntranet(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestIntranetIP(t *testing.T) {
	// 简单测试不返回错误
	ips, err := IntranetIP()
	if err != nil {
		t.Errorf("IntranetIP() error = %v", err)
		return
	}
	// 可能返回空（在无内网环境），但不应当报错
	t.Logf("IntranetIP() returned %d IPs: %v", len(ips), ips)
}
