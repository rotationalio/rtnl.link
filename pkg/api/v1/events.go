package api

type Click struct {
	Time      string `json:"time"`
	Views     int    `json:"views"`
	UserAgent string `json:"user-agent"`
	IPAddr    string `json:"ip_address"`
}
