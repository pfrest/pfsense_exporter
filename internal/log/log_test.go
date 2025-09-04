package log

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func captureOutput(f func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	f()
	return buf.String()
}

func TestDebug(t *testing.T) {
	// Test with verbose enabled
	originalVerbose := Verbose
	defer func() { Verbose = originalVerbose }()

	Verbose = true
	output := captureOutput(func() {
		Debug("test-scope", "test message %s", "arg1")
	})

	if !strings.Contains(output, "level='DEBUG'") {
		t.Error("Expected DEBUG level in output")
	}
	if !strings.Contains(output, "scope='test-scope'") {
		t.Error("Expected scope in output")
	}
	if !strings.Contains(output, "test message arg1") {
		t.Error("Expected formatted message in output")
	}

	// Test with verbose disabled
	Verbose = false
	output = captureOutput(func() {
		Debug("test-scope", "test message")
	})

	if output != "" {
		t.Error("Expected no output when verbose is disabled")
	}
}

func TestInfo(t *testing.T) {
	output := captureOutput(func() {
		Info("test-scope", "test message %s %d", "arg1", 42)
	})

	if !strings.Contains(output, "level='INFO'") {
		t.Error("Expected INFO level in output")
	}
	if !strings.Contains(output, "scope='test-scope'") {
		t.Error("Expected scope in output")
	}
	if !strings.Contains(output, "test message arg1 42") {
		t.Error("Expected formatted message in output")
	}
}

func TestWarn(t *testing.T) {
	output := captureOutput(func() {
		Warn("test-scope", "warning message %s", "warn1")
	})

	if !strings.Contains(output, "level='WARN'") {
		t.Error("Expected WARN level in output")
	}
	if !strings.Contains(output, "scope='test-scope'") {
		t.Error("Expected scope in output")
	}
	if !strings.Contains(output, "warning message warn1") {
		t.Error("Expected formatted message in output")
	}
}

func TestError(t *testing.T) {
	output := captureOutput(func() {
		Error("test-scope", "error message %s", "error1")
	})

	if !strings.Contains(output, "level='ERROR'") {
		t.Error("Expected ERROR level in output")
	}
	if !strings.Contains(output, "scope='test-scope'") {
		t.Error("Expected scope in output")
	}
	if !strings.Contains(output, "error message error1") {
		t.Error("Expected formatted message in output")
	}
}

func TestFatal(t *testing.T) {
	// Testing Fatal is tricky since it calls log.Fatalf which exits the program
	// We'll test it indirectly by checking its behavior in a subprocess

	if os.Getenv("TEST_FATAL") == "1" {
		// This will only run in the subprocess
		Fatal("test-scope", "fatal message %s", "arg1")
		return
	}

	// Run the test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "TEST_FATAL=1")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Fatal should cause the subprocess to exit with non-zero status
	if err == nil {
		t.Error("Expected Fatal to cause process exit")
	}

	// Check that the fatal message was logged
	output := stderr.String()
	if !strings.Contains(output, "level='FATAL'") {
		t.Error("Expected FATAL level in output")
	}
	if !strings.Contains(output, "scope='test-scope'") {
		t.Error("Expected scope in output")
	}
	if !strings.Contains(output, "fatal message arg1") {
		t.Error("Expected formatted message in output")
	}
}

func TestVerboseFlag(t *testing.T) {
	originalVerbose := Verbose
	defer func() { Verbose = originalVerbose }()

	// Test setting verbose to true
	Verbose = true
	if !Verbose {
		t.Error("Expected Verbose to be true")
	}

	// Test setting verbose to false
	Verbose = false
	if Verbose {
		t.Error("Expected Verbose to be false")
	}
}
