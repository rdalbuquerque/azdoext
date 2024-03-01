package logger

import (
	"fmt"
	"os"
)

type Logger struct {
	filename string
}

func NewLogger(filename string) *Logger {
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
