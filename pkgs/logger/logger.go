package logger

import (
	"fmt"
	"os"
	"path/filepath"
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
	f, err := os.OpenFile(l.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("[%s] %s\n", identifier, msg))
}
