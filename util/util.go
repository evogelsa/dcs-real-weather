package util

// Must performs a lazy error "check"
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// Configuration is the structure of config.json to be parsed
type Configuration struct {
	APIKey     string `json:"api-key"`
	ICAO       string `json:"icao"`
	TimeOffset int    `json:"time-offset"`
}
