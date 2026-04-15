package uAddress

import "net"

// GetLocalIp 获取本机第一个非回环 IPv4 地址（兼容旧 API）
//
// 优先使用 IntranetIP 的结果，如果无法获取非回环地址，则返回 "127.0.0.1"
//
// 使用示例：
//
//	ip := uAddress.GetLocalIp()
//	// ip = "192.168.1.100"
func GetLocalIp() string {
	ips, err := IntranetIP()
	if err == nil && len(ips) > 0 {
		return ips[0]
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			ipv4 := ipnet.IP.To4()
			if ipv4 != nil {
				return ipv4.String()
			}
		}
	}
	return "127.0.0.1"
}
