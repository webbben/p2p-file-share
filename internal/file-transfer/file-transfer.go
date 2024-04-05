package filetransfer

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/webbben/p2p-file-share/internal/config"
	"github.com/webbben/p2p-file-share/internal/model"
	"github.com/webbben/p2p-file-share/internal/network"
)

const (
	port = 8080
)

// sends a file to another node
func SendFile(conn net.Conn, filePath string) (bool, error) {
	defer conn.Close()

	mountDir := config.GetMountDir(nil) // TODO pass config in instead of loading it
	// open the file
	file, err := os.Open(filepath.Join(mountDir, filePath))
	if err != nil {
		_, err := conn.Write([]byte("ERROR: Failed to open file: " + filePath))
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
	mountDir := config.GetMountDir(nil) // TODO pass in the config instead of loading it each time
	fullPath := filepath.Join(mountDir, filePath)
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// read an initial buffer to check for error messages
	buf, err := network.ReadBuffer(conn, 1024)
	if err != nil {
		os.Remove(fullPath)
		return err
	}
	b1, err := file.Write(buf)
	if err != nil {
		os.Remove(fullPath)
		return err
	}
	// Read any remaining data in the stream
	b2, err := io.Copy(file, conn)
	if err != nil {
		os.Remove(fullPath)
		return err
	}
	fmt.Printf("wrote %v bytes to %s\n", int64(b1)+b2, fullPath)
	return nil
}
