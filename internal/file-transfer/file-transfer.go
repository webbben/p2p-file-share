package filetransfer

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/webbben/p2p-file-share/internal/config"
	"github.com/webbben/p2p-file-share/internal/model"
)

const (
	port = 8080
)

// sends a file to another node
func SendFile(conn net.Conn, filePath string) (bool, error) {
	defer conn.Close()

	// open the file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error sending file:", err)
		return false, err
	}
	defer file.Close()

	// send the file
	_, err = io.Copy(conn, file)
	if err != nil {
		fmt.Println("Error sending file:", err)
		return false, err
	}
	fmt.Println("File sent successfully!")
	return true, nil
}

// requests a file from another node
func RequestFile(senderIP string, filePath string) (bool, error) {
	// connect to the sender node
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%v", senderIP, port))
	if err != nil {
		fmt.Println("Error receiving file:", err)
		return false, err
	}
	defer conn.Close()

	req := model.FileRequest{
		Type: config.TYPE_FILE_REQUEST,
		File: filePath,
	}
	reqJson, err := json.Marshal(req)
	if err != nil {
		return false, err
	}

	// send the file request info to the other node
	_, err = conn.Write(reqJson)
	if err != nil {
		return false, err
	}

	// TODO: receive the file over the existing connection
	err = receiveFile(conn, filePath)
	if err != nil {
		fmt.Println("error receiving file:", err)
		return false, err
	}
	fmt.Println("received file:", filePath)
	return true, nil
}

func receiveFile(conn net.Conn, filePath string) error {
	// Create or open the file for writing
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read data from the connection and write it to the file
	_, err = io.Copy(file, conn)
	if err != nil {
		return err
	}
	return nil
}
