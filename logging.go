package logruswrapper

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	log  *logrus.Logger
	once sync.Once
)

type Fields = logrus.Fields

func init() {
	log = logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})
}

func Setup(level string, isProduction bool) {
	once.Do(func() {
		lvl, err := logrus.ParseLevel(level)
		if err != nil {
			lvl = logrus.InfoLevel
		}
		log.SetLevel(lvl)

		if isProduction {
			log.SetFormatter(&logrus.JSONFormatter{
				TimestampFormat: time.RFC3339,
			})
		} else {
			log.SetFormatter(&logrus.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: time.RFC3339,
				ForceColors:     true,
			})
		}
	})
}

func generateLogger(ctx context.Context, fields *Fields) *logrus.Entry {
	entry := log.WithFields(*fields)
	if fields != nil {
		entry.WithFields(*fields)
	}

	return entry.WithContext(ctx)
}

func getCaller() *logrus.Fields {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return nil
	}

	fnName := runtime.FuncForPC(pc).Name()
	if lastSlash := strings.LastIndex(file, "/"); lastSlash >= 0 {
		file = file[lastSlash+1:]
	}

	fields := logrus.Fields{
		"file": fmt.Sprintf("%s:%d", file, line),
		"func": fnName,
	}

	return &fields
}

func Info(ctx context.Context, msg string, fields *Fields) {
	callerFields := getCaller()
	generateLogger(ctx, fields).WithFields(*callerFields).Info(msg)
}

func Error(ctx context.Context, msg string, err error, fields *Fields) {
	callerFields := getCaller()
	generateLogger(ctx, fields).WithFields(*callerFields).WithError(err).Error(msg)
}

func Debug(ctx context.Context, msg string, fields *Fields) {
	callerFields := getCaller()
	generateLogger(ctx, fields).WithFields(*callerFields).Debug(msg)
}

func Warn(ctx context.Context, msg string, fields *Fields) {
	callerFields := getCaller()
	generateLogger(ctx, fields).WithFields(*callerFields).Warn(msg)
}

func Fatal(ctx context.Context, msg string, fields *Fields) {
	callerFields := getCaller()
	generateLogger(ctx, fields).WithFields(*callerFields).Fatal(msg)
}
