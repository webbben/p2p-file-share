package model

/*
All Request types should include a "type" property so the TCP servers can detect the purpose of the message
*/

type Handshake struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// a request for a file to be sent from one node to another
type FileRequest struct {
	Type string `json:"type"`
	File string `json:"file"`
}
