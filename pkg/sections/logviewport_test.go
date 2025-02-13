package sections

import (
	"os"
	"testing"
)

func TestFormatLog(t *testing.T) {
	f, err := os.Open("testdata/log-test.txt")
	if err != nil {
		t.Fatalf("unable to open original file: %v", err)
	}
	defer f.Close()

	formattedLog := formatLog(f)
	expectedFormattedLog, err := os.ReadFile("testdata/expected-formatted-log-test.txt")
	if err != nil {
		t.Fatalf("unable to read expected formatted log")
	}
	if formattedLog != string(expectedFormattedLog) {
		t.Fatalf("formatted log is not as expected: %s", formattedLog)
	}
}
