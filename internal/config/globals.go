package config

// this file defines global values for the whole application to consume

const (
	PORT int = 8080
)

// message types
const (
	TYPE_DISCOVER_PEER string = "discover_peer"
	TYPE_FILE_REQUEST  string = "file_request"
)

const (
	MESSAGE_TIMEOUT_MS      int = 1000 // duration in ms until tcp connection should timeout
	MESSAGE_TIMEOUT_MS_LONG int = 5000 // a longer duration in ms to wait until timing out tcp connection
)
