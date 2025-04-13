package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Make the SSE request
	resp, err := http.Get("http://localhost:8080/events")
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Create a reader for the response body
	reader := bufio.NewReader(resp.Body)

	// Print headers
	fmt.Println("Response Headers:")
	for k, v := range resp.Header {
		fmt.Printf("%s: %v\n", k, v)
	}
	fmt.Println("\nEvents:")

	// Read and print events
	for {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, closing connection...")
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Println("Connection closed by server")
					return
				}
				fmt.Printf("Error reading response: %v\n", err)
				return
			}
			fmt.Print(line)
		}
	}
}
