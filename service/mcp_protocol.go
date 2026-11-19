package service

func MCPProtocolSupported(version string) bool {
	return version == "2025-11-25" || version == "2025-06-18" || version == "2025-03-26"
}
