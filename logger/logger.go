package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log *zap.SugaredLogger

// Init initializes the logger after the config file has been read
func Init(
	filename string,
	maxSize int,
	maxBackups int,
	maxAge int,
	compress bool,
	level zapcore.Level,
) {
	var core zapcore.Core

	consoleEncoderConfig := zapcore.EncoderConfig{
		TimeKey:      "time",
		LevelKey:     "level",
		CallerKey:    "caller",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.CapitalColorLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

	consoleWriter := zapcore.AddSync(os.Stdout)

	if filename != "" {
		lumberjackLogger := &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		}

		fileEncoderConfig := zapcore.EncoderConfig{
			TimeKey:      "time",
			LevelKey:     "level",
			CallerKey:    "caller",
			MessageKey:   "message",
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeLevel:  zapcore.CapitalLevelEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		}

		fileEncoder := zapcore.NewConsoleEncoder(fileEncoderConfig)

		fileWriter := zapcore.AddSync(lumberjackLogger)

		// create a new zapcore using both outputs
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleWriter, level),
			zapcore.NewCore(fileEncoder, fileWriter, level),
		)
	} else {
		core = zapcore.NewCore(consoleEncoder, consoleWriter, level)
	}

	// Create a Sugared Logger from the core
	if level == zapcore.DebugLevel {
		// add caller if debug level
		log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
	} else {
		log = zap.New(core).Sugar()
	}
}

func Debugf(template string, args ...any) {
	log.Debugf(template, args...)
}

func Debugln(args ...any) {
	log.Debugln(args...)
}

func Infof(template string, args ...any) {
	log.Infof(template, args...)
}

func Infoln(args ...any) {
	log.Infoln(args...)
}

func Infow(msg string, keysAndValues ...any) {
	log.Infow(msg, keysAndValues...)
}

func Warnf(template string, args ...any) {
	log.Warnf(template, args...)
}

func Warnln(args ...any) {
	log.Warnln(args...)
}

func Errorf(template string, args ...any) {
	log.Errorf(template, args...)
}

func Errorln(args ...any) {
	log.Errorln(args...)
}

func Fatalf(template string, args ...any) {
	log.Fatalf(template, args...)
}

func Fatalln(args ...any) {
	log.Fatalln(args...)
}
