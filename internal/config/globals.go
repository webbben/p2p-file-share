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
