package go_tools

import (
	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type GroupLog struct {
	Caption string
	Logs    [][]zap.Field
}

type OnCall func()

type Logger struct {
	logger         *zap.Logger
	prevTime       time.Time
	waitGroup      sync.WaitGroup
	debug          bool
	outputDir      string
	errorOutputDir string
	traceLevel     int
	key            string
	caption        string
	group          [][]zap.Field
	OnCall         OnCall
}

func (this *Logger) SetOnCall(call OnCall) {
	this.OnCall = call

}
func (this *Logger) SetDebug(status bool) {
	this.debug = status

}

func (this *Logger) SetLevel(level int) {
	this.traceLevel = level
}

func (this *Logger) Debug(msg string, fields ...zap.Field) {
	this.rotate()
	field := make([]zap.Field, 1)
	field[0] = zap.String("caller", this.getCaller(2))
	fields = append(field, fields...)
	this.logger.Debug(msg, fields...)
	if this.OnCall != nil {
		this.OnCall()
	}
}

func (this *Logger) InfoSimple(fields ...zap.Field) {
	this.rotate()
	fields = append([]zap.Field{zap.Any("prev", this.getCaller(3))}, fields...)
	this.logger.Info(this.getCaller(2), fields...)
	if this.OnCall != nil {
		this.OnCall()
	}
}

func (this *Logger) Exception(object interface{}) {
	for i := 0; i < 10; i++ {
		this.logger.Info(this.getCaller(i), zap.Any(strconv.Itoa(i), object))
	}
}

func (this *Logger) OnIncoming(msg []byte) {
	this.rotate()
	this.logger.Info("incoming", zap.Any("message", string(msg)))
}

func (this *Logger) OnOutgoing(msg []byte) {
	this.rotate()
	this.logger.Info("outcoming", zap.Any("message", string(msg)))
}

func (this *Logger) OnEvent(msg string) {
	this.rotate()
	this.logger.Info("event", zap.Any("message", msg))
}

func (this *Logger) OnEventf(msg string, data ...interface{}) {
	this.rotate()
	this.logger.Info("event", zap.Any("message", fmt.Sprintf(msg, data...)))
}

func (this *Logger) DebugSimple(fields ...zap.Field) {
	if this.debug {
		this.rotate()
		this.logger.Info(this.getCaller(2), fields...)
	}
	if this.OnCall != nil {
		this.OnCall()
	}
}
func (this *Logger) Trace(level int, fields ...zap.Field) {
	if this.traceLevel >= level {
		this.rotate()
		this.logger.Info(this.getCaller(2), fields...)
	}
	if this.OnCall != nil {
		this.OnCall()
	}
}

// Info logs a message at InfoLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (this *Logger) Info(msg string, fields ...zap.Field) {
	this.rotate()
	field := make([]zap.Field, 1)
	field[0] = zap.String("caller", this.getCaller(2))
	fields = append(field, fields...)
	this.logger.Info(msg, fields...)
	if this.OnCall != nil {
		this.OnCall()
	}
}

// Warn logs a message at WarnLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (this *Logger) Warn(msg string, fields ...zap.Field) {
	this.rotate()
	field := make([]zap.Field, 1)
	field[0] = zap.String("caller", this.getCaller(2))
	fields = append(field, fields...)
	this.logger.Warn(msg, fields...)
	if this.OnCall != nil {
		this.OnCall()
	}
}

// Error logs a message at ErrorLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func (this *Logger) Error(msg string, fields ...zap.Field) {
	this.rotate()
	field := make([]zap.Field, 1)
	field[0] = zap.String("caller", this.getCaller(2))
	fields = append(field, fields...)
	this.logger.Error(msg, fields...)
	if this.OnCall != nil {
		this.OnCall()
	}
}

func (this *Logger) getCaller(skipframe int) string {
	frame := this.getFrame(skipframe)
	return frame.Function + ":" + strconv.Itoa(frame.Line)
	//return frame.File + " (" + frame.Function + ":" + strconv.Itoa(frame.Line) + ")"
}

func (this *Logger) getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

func (this *Logger) rotate() {
	this.waitGroup.Add(1)
	if strings.Compare(this.prevTime.Format("2006-01-02"), time.Now().Format("2006-01-02")) != 0 {
		this.prevTime = time.Now()
		OutputPaths := []string{this.outputDir + time.Now().Format("2006-01-02") + ".log"}
		ErrorOutputPaths := []string{this.errorOutputDir + time.Now().Format("2006-01-02") + ".log"}
		if this.debug {
			OutputPaths = append(OutputPaths, "stdout")
			ErrorOutputPaths = append(ErrorOutputPaths, "stderr")
		}
		logger, e := LoggerGenerator(OutputPaths, ErrorOutputPaths)
		if e == nil {
			this.logger = logger
		}
	}
	this.waitGroup.Done()
}

func LoggerRotateGenerator(OutputDir string, ErrorOutputDir string, prefix string, debug bool) (*Logger, error) {
	e := os.MkdirAll(OutputDir, os.ModePerm)
	e = os.MkdirAll(ErrorOutputDir, os.ModePerm)
	OutputPaths := []string{filepath.Join(OutputDir, prefix+"-"+time.Now().Format("2006-01-02")+".log")}
	ErrorOutputPaths := []string{filepath.Join(ErrorOutputDir, prefix+"-"+time.Now().Format("2006-01-02")+".log")}
	if debug {
		OutputPaths = append(OutputPaths, "stdout")
		ErrorOutputPaths = append(ErrorOutputPaths, "stderr")
	}
	logger, e := LoggerGeneratorExcCaller(OutputPaths, ErrorOutputPaths)
	if e != nil {
		return nil, e
	} else {
		return &Logger{logger: logger, prevTime: time.Now(), debug: debug, outputDir: filepath.Join(OutputDir, prefix+"-"), errorOutputDir: filepath.Join(ErrorOutputDir, prefix+"-")}, nil
	}
}
func LoggerRotateGeneratorSimple(OutputDir string, ErrorOutputDir string, prefix string, debug bool) (*Logger, error) {
	e := os.MkdirAll(OutputDir, os.ModePerm)
	e = os.MkdirAll(ErrorOutputDir, os.ModePerm)
	OutputPaths := []string{filepath.Join(OutputDir, prefix+"-"+time.Now().Format("2006-01-02")+".log")}
	ErrorOutputPaths := []string{filepath.Join(ErrorOutputDir, prefix+"-"+time.Now().Format("2006-01-02")+".log")}
	if debug {
		OutputPaths = append(OutputPaths, "stdout")
		ErrorOutputPaths = append(ErrorOutputPaths, "stderr")
	}
	logger, e := LoggerGeneratorSimple(OutputPaths, ErrorOutputPaths)
	if e != nil {
		return nil, e
	} else {
		return &Logger{logger: logger, prevTime: time.Now(), debug: debug, outputDir: filepath.Join(OutputDir, prefix+"-"), errorOutputDir: filepath.Join(ErrorOutputDir, prefix+"-")}, nil
	}
}

func LoggerRotateGroupGeneratorSimple(OutputDir string, ErrorOutputDir string, prefix string, debug bool, key string, caption string) (*Logger, error) {
	e := os.MkdirAll(OutputDir, os.ModePerm)
	e = os.MkdirAll(ErrorOutputDir, os.ModePerm)
	OutputPaths := []string{filepath.Join(OutputDir, prefix+"-"+time.Now().Format("2006-01-02")+".log")}
	ErrorOutputPaths := []string{filepath.Join(ErrorOutputDir, prefix+"-"+time.Now().Format("2006-01-02")+".log")}
	if debug {
		OutputPaths = append(OutputPaths, "stdout")
		ErrorOutputPaths = append(ErrorOutputPaths, "stderr")
	}
	logger, e := LoggerGeneratorSimple(OutputPaths, ErrorOutputPaths)
	if e != nil {
		return nil, e
	} else {
		return &Logger{logger: logger, prevTime: time.Now(), debug: debug, outputDir: filepath.Join(OutputDir, prefix+"-"), errorOutputDir: filepath.Join(ErrorOutputDir, prefix+"-")}, nil
	}
}

func LoggerGenerator(OutputPaths []string, ErrorOutputPaths []string) (*zap.Logger, error) {
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		OutputPaths:      OutputPaths,
		ErrorOutputPaths: ErrorOutputPaths,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "info",
			LevelKey:     "level",
			EncodeLevel:  zapcore.LowercaseLevelEncoder,
			TimeKey:      "time",
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("logger construction succeeded")
	return logger, err
}

func LoggerGeneratorExcCaller(OutputPaths []string, ErrorOutputPaths []string) (*zap.Logger, error) {
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		OutputPaths:      OutputPaths,
		ErrorOutputPaths: ErrorOutputPaths,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "info",
			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
			TimeKey:     "time",
			EncodeTime:  zapcore.ISO8601TimeEncoder,
		},
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("logger construction succeeded")
	return logger, err
}

func LoggerGeneratorSimple(OutputPaths []string, ErrorOutputPaths []string) (*zap.Logger, error) {
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		OutputPaths:      OutputPaths,
		ErrorOutputPaths: ErrorOutputPaths,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "info",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
			TimeKey:     "time",
			EncodeTime:  zapcore.ISO8601TimeEncoder,
		},
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("logger construction succeeded")
	return logger, err
}

func Ekstration(pathFlac string, nameFlac string, target ...interface{}) error {
	configPath := flag.String(pathFlac, ".", "Configuration YAML path")
	configName := flag.String(nameFlac, "config-prod", "Configuration Name { config-prod | config-dev } (Required)")
	flag.Parse()

	// config file path
	viper.AddConfigPath(*configPath)
	// config file name
	viper.SetConfigName(*configName)

	err := viper.ReadInConfig()

	if err != nil {
		return err
	}

	for _, element := range target {
		err = viper.Unmarshal(&element)
		if err != nil {
			return err
		}
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("config changed")
		for _, element := range target {
			//element = ""
			err = viper.Unmarshal(&element)
		}
	})

	return nil
}
