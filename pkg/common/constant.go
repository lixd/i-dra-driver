package common

import "time"

const (
	DriverName   string = "gopher.example.com"
	ResourceName string = "gopher.example.com/gopher"
	DevicePath   string = "/etc/gophers"
	CDIVendor    string = "gopher.example.com"
	CDIClass     string = "gopher"

	RescanInterval = 60 * time.Second
)
