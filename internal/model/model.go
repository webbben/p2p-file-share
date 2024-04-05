package model

// a request for a file to be sent from one node to another
type FileRequest struct {
	Type string `json:"type"`
	File string `json:"file"`
}
