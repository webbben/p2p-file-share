package config

// this file defines global values for the whole application to consume

const (
	PORT int = 8080
)

// message types
const (
	// message meant for discovering a peer node
	TYPE_DISCOVER_PEER string = "discover_peer"
	// message for requesting a file from another node
	TYPE_FILE_REQUEST string = "file_request"
	// message for notifying another node of a file change
	TYPE_FILE_CHANGE_NOTIFY string = "file_change_notify"
	// message from a node that wants to scan this node's files
	TYPE_SCAN_FILES string = "scan_files"
)

const (
	MESSAGE_TIMEOUT_MS      int = 1000 // duration in ms until tcp connection should timeout
	MESSAGE_TIMEOUT_MS_LONG int = 5000 // a longer duration in ms to wait until timing out tcp connection
)
