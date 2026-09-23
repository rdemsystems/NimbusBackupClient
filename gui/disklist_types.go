package main

// PhysicalDiskInfo represents information about a physical disk.
//
// Kept in an untagged file: disklist_windows.go / disklist_linux.go use it and
// are compiled into the service build too, where main.go (!service) is not.
type PhysicalDiskInfo struct {
	DiskNumber   int64  `json:"disk_number"`
	Size         int64  `json:"size"`
	Model        string `json:"model"`
	IsBootDisk   bool   `json:"is_boot_disk"`
	IsSystemDisk bool   `json:"is_system_disk"`
	DeviceID     string `json:"device_id"`
	DevicePath   string `json:"device_path"`
}
