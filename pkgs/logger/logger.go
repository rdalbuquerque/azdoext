package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	filename string
}

func NewLogger(filename string) *Logger {
	// if OS is Windows, concatenate LOCALAPPDATA environment variable to filename
	if os.Getenv("OS") == "Windows_NT" {
		filename = os.Getenv("LOCALAPPDATA") + "\\azdoext\\" + filename
	}
	dir := filepath.Dir(filename)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return &Logger{
		filename: filename,
	}
}

func (l *Logger) LogToFile(identifier, msg string) {
	if os.Getenv("DEBUG_AZDOEXT") == "" {
		return
	}
	f, err := os.OpenFile(l.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("%s | %s | %s\n", timestamp, identifier, msg))
}
