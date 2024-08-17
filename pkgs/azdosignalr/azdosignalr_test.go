package azdosignalr

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"
)

func TestStartReceivingLoop(t *testing.T) {
	signalrConn, err := NewSignalRConn("rdalbuquerque", "accountIDPlaceholder", "explore-bubbletea")
	if err != nil {
		t.Errorf("NewSignalRClient() failed: %v", err)
	}
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Start the receiving loop in a separate goroutine
	// go signalrConn.StartReceivingLoop()

	// Allow some time for the message to be processed
	time.Sleep(10 * time.Second) // Adjust the sleep duration as needed

	// Close the connection
	signalrConn.Conn.Close()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check if any messages were printed to stdout
	if output == "" {
		t.Errorf("No messages were printed to stdout")
	} else {
		t.Logf("Captured output:\n%s", output)
	}
}

func TestSendMessage(t *testing.T) {
	signalrConn, err := NewSignalRConn("rdalbuquerque", "accountIDPlaceholder", "explore-bubbletea")
	if err != nil {
		t.Errorf("NewSignalRClient() failed: %v", err)
	}

	// Send a message
	err = signalrConn.SendMessage("chat", "send", []interface{}{"Hello, SignalR!"})
	if err != nil {
		t.Errorf("SendMessage() failed: %v", err)
	}
}

func TestSendBuildSelection(t *testing.T) {
	project := "160c770b-e289-4b64-9b14-af7475a1b744"
	signalrConn, err := NewSignalRConn("rdalbuquerque", "accountIDPlaceholder", project)
	if err != nil {
		t.Errorf("NewSignalRClient() failed: %v", err)
	}

	// Start the receiving loop in a separate goroutine
	// go signalrConn.StartReceivingLoop()

	// Select a build to watch
	err = signalrConn.SendMessage("builddetailhub", "WatchBuild", []interface{}{project, 887})
	if err != nil {
		t.Errorf("SendMessage() failed: %v", err)
	}

	// Allow some time for the messages to be received
	time.Sleep(20 * time.Second)

	// Close the connection
	signalrConn.Conn.Close()
}
