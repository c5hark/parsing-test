package lenta

type Config struct {
	ProxyURL      string
	SessionToken  string
	StoreID       int
	Domain        string
	DeviceID      string
	UserSessionID string
	OutputPath    string
	CategoryIDs   []int
}
