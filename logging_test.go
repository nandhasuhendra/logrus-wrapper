package logruswrapper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

// resetOnce resets the sync.Once so Setup can be called again in tests.
func resetOnce() {
	once = sync.Once{}
}

// captureOutput redirects the logger output to a buffer and returns it.
func captureOutput() *bytes.Buffer {
	buf := &bytes.Buffer{}
	log.SetOutput(buf)
	log.SetFormatter(&logrus.JSONFormatter{})
	return buf
}

// restoreOutput restores logger output to os.Stdout.
func restoreOutput() {
	log.SetOutput(nil) // logrus falls back to os.Stderr when nil; restored via os.Stdout below
	log.SetFormatter(&logrus.JSONFormatter{})
}

func TestSetup_DevelopmentMode(t *testing.T) {
	resetOnce()
	defer resetOnce()

	Setup("debug", false)

	if log.GetLevel() != logrus.DebugLevel {
		t.Errorf("expected level %v, got %v", logrus.DebugLevel, log.GetLevel())
	}

	if _, ok := log.Formatter.(*logrus.TextFormatter); !ok {
		t.Error("expected TextFormatter in development mode")
	}
}

func TestSetup_ProductionMode(t *testing.T) {
	resetOnce()
	defer resetOnce()

	Setup("warn", true)

	if log.GetLevel() != logrus.WarnLevel {
		t.Errorf("expected level %v, got %v", logrus.WarnLevel, log.GetLevel())
	}

	if _, ok := log.Formatter.(*logrus.JSONFormatter); !ok {
		t.Error("expected JSONFormatter in production mode")
	}
}

func TestSetup_InvalidLevel_DefaultsToInfo(t *testing.T) {
	resetOnce()
	defer resetOnce()

	Setup("notavalidlevel", true)

	if log.GetLevel() != logrus.InfoLevel {
		t.Errorf("expected InfoLevel as default, got %v", log.GetLevel())
	}
}

func TestSetup_OnlyAppliesOnce(t *testing.T) {
	resetOnce()
	defer resetOnce()

	Setup("debug", false)
	Setup("error", true) // second call must be ignored

	if log.GetLevel() != logrus.DebugLevel {
		t.Errorf("second Setup call must be a no-op; expected DebugLevel, got %v", log.GetLevel())
	}
}

func TestInfo(t *testing.T) {
	buf := captureOutput()
	defer restoreOutput()
	log.SetLevel(logrus.InfoLevel)

	ctx := context.Background()
	fields := Fields{"key": "value"}
	Info(ctx, "info message", &fields)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON output: %v", err)
	}

	if entry["msg"] != "info message" {
		t.Errorf("expected msg 'info message', got %v", entry["msg"])
	}
	if entry["level"] != "info" {
		t.Errorf("expected level 'info', got %v", entry["level"])
	}
	if entry["key"] != "value" {
		t.Errorf("expected field key='value', got %v", entry["key"])
	}
	if _, ok := entry["file"]; !ok {
		t.Error("expected 'file' caller field in output")
	}
	if _, ok := entry["func"]; !ok {
		t.Error("expected 'func' caller field in output")
	}
}

func TestError(t *testing.T) {
	buf := captureOutput()
	defer restoreOutput()
	log.SetLevel(logrus.ErrorLevel)

	ctx := context.Background()
	fields := Fields{"request_id": "abc-123"}
	Err := errors.New("something went wrong")
	Error(ctx, "error message", &fields, Err)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON output: %v", err)
	}

	if entry["msg"] != "error message" {
		t.Errorf("expected msg 'error message', got %v", entry["msg"])
	}
	if entry["level"] != "error" {
		t.Errorf("expected level 'error', got %v", entry["level"])
	}
	if entry["error"] != "something went wrong" {
		t.Errorf("expected error field, got %v", entry["error"])
	}
	if entry["request_id"] != "abc-123" {
		t.Errorf("expected request_id field, got %v", entry["request_id"])
	}
}

func TestDebug(t *testing.T) {
	buf := captureOutput()
	defer restoreOutput()
	log.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	fields := Fields{"component": "worker"}
	Debug(ctx, "debug message", &fields)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON output: %v", err)
	}

	if entry["msg"] != "debug message" {
		t.Errorf("expected msg 'debug message', got %v", entry["msg"])
	}
	if entry["level"] != "debug" {
		t.Errorf("expected level 'debug', got %v", entry["level"])
	}
}

func TestWarn(t *testing.T) {
	buf := captureOutput()
	defer restoreOutput()
	log.SetLevel(logrus.WarnLevel)

	ctx := context.Background()
	fields := Fields{"threshold": 90}
	Warn(ctx, "warn message", &fields)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON output: %v", err)
	}

	if entry["msg"] != "warn message" {
		t.Errorf("expected msg 'warn message', got %v", entry["msg"])
	}
	if entry["level"] != "warning" {
		t.Errorf("expected level 'warning', got %v", entry["level"])
	}
}

func TestDebug_SuppressedWhenLevelIsInfo(t *testing.T) {
	buf := captureOutput()
	defer restoreOutput()
	log.SetLevel(logrus.InfoLevel)

	ctx := context.Background()
	fields := Fields{}
	Debug(ctx, "should not appear", &fields)

	if strings.TrimSpace(buf.String()) != "" {
		t.Errorf("expected no output for debug when level is info, got: %s", buf.String())
	}
}

func TestGetCaller_ReturnsFileAndFunc(t *testing.T) {
	// getCaller uses depth=2: direct call here simulates one extra frame.
	// We call it indirectly through a wrapper to match production depth.
	wrapper := func() *Fields {
		return getCaller()
	}
	fields := wrapper()

	if fields == nil {
		t.Fatal("expected non-nil fields from getCaller")
	}

	file, ok := (*fields)["file"].(string)
	if !ok || file == "" {
		t.Error("expected non-empty 'file' field")
	}
	fn, ok := (*fields)["func"].(string)
	if !ok || fn == "" {
		t.Error("expected non-empty 'func' field")
	}
}

func TestGenerateLogger_ContextPropagated(t *testing.T) {
	type ctxKey string
	const key ctxKey = "trace_id"

	ctx := context.WithValue(context.Background(), key, "abc-xyz")
	fields := Fields{"svc": "auth"}

	entry := generateLogger(ctx, &fields)

	if entry.Context == nil {
		t.Fatal("expected context to be attached to log entry")
	}
	if got := entry.Context.Value(key); got != "abc-xyz" {
		t.Errorf("expected trace_id 'abc-xyz' in context, got %v", got)
	}
}
