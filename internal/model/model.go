package model

import "fmt"

/*
All messages should include a "type" property so the TCP servers can detect the purpose of the message
*/

type Handshake struct {
	Type     string `json:"type"`
	Data     string `json:"data"`     // misc data to send in the handshake, in case we want to verify authenticity (TODO)
	Nickname string `json:"nickname"` // nickname of the node sending this handshake
}

// a request for a file to be sent from one node to another
type FileRequest struct {
	Type string `json:"type"`
	File string `json:"file"` // the path of the file (relative to the mount directory)
}

type NotifyFileChange struct {
	Type   string `json:"type"`
	File   string `json:"file"`   // the path of the file (relative to the mount directory)
	Change string `json:"change"` // the type of change that occurred, e.g. modified, deleted, etc.
}

func (n NotifyFileChange) String() string {
	return fmt.Sprintf("%s (%s)", n.File, n.Change)
}

type Peer struct {
	IP       string `json:"ip"`
	Nickname string `json:"nickname"`
}
