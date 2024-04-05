package main

/*
Intended for creating CLI tools for testing specific functions.
*/

import (
	"flag"
	"fmt"
	"os"

	filetransfer "github.com/webbben/p2p-file-share/internal/file-transfer"
)

func main() {
	// Define flags
	requestFileCmd := flag.NewFlagSet("requestFile", flag.ExitOnError)
	// Define flags specific to the "test" command
	reqFileArg := requestFileCmd.String("file", "", "file to request")
	reqIpArg := requestFileCmd.String("ip", "", "IP of node to request file from")

	// Parse command-line arguments
	if len(os.Args) < 2 {
		fmt.Println("Error: Please provide a command to test")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "requestFile":
		requestFileCmd.Parse(os.Args[2:])
		if *reqFileArg == "" {
			fmt.Println("Error: file argument is required for requestFile command")
			requestFileCmd.Usage()
			os.Exit(1)
		}
		if *reqIpArg == "" {
			fmt.Println("Error: ip argument is required for requestFile command")
			os.Exit(1)
		}
		fmt.Printf("Requesting file %s from node %s\n", *reqFileArg, *reqIpArg)
		filetransfer.RequestFile(*reqIpArg, *reqFileArg)
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}
