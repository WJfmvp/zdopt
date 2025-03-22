package Logs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Level int

const (
	// 定义日志级别
	Debug Level = iota
	Info  Level = iota
	Warn
	Error
	Fatal
)

var (
	logDir      = "logs"
	logDirMutex sync.Mutex
)

type Logger struct {
	*log.Logger
	mu     sync.Mutex
	level  Level
	writer logWriter
}

type logWriter interface {
	Write(p []byte) (n int, err error)
	Close() error
}

// NewLogger 创建新的Logger实例（带错误返回）
func NewLogger(level Level, loggerName string) (*Logger, error) {
	baseLogger, writer, err := createLogger(loggerName)
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger: baseLogger,
		level:  level,
		writer: writer,
	}, nil
}

func createLogger(loggerName string) (*log.Logger, logWriter, error) {
	if loggerName == "" {
		return log.New(os.Stdout, "", log.LstdFlags), nil, nil
	}

	if err := ensureLogDir(); err != nil {
		return nil, nil, err
	}

	file, err := openLogFile(loggerName)
	if err != nil {
		return nil, nil, err
	}

	return log.New(file, fmt.Sprintf("[%s] ", loggerName), log.Ldate|log.Ltime|log.Lshortfile), file, nil
}

func ensureLogDir() error {
	logDirMutex.Lock()
	defer logDirMutex.Unlock()

	_, err := os.Stat(logDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
	}
	return nil
}

func (l Level) String() string {
	return [...]string{"INFO", "WARN", "ERROR", "FATAL"}[l]
}

// CreateConsoleLogConfig 创建控制台日志配置
func CreateConsoleLogConfig(loggerName string) *log.Logger {
	// 创建带有自定义设置的日志器
	return log.New(os.Stdout, fmt.Sprintf("[%s] ", loggerName), log.Ldate|log.Ltime|log.Lshortfile)
}

// CreateFileLogConfig 创建文件日志配置
func CreateFileLogConfig(loggerName string) *log.Logger {
	file, err := os.OpenFile(fmt.Sprintf("logs/%s.log", loggerName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("打开文件错误:", err)
	}
	return log.New(file, fmt.Sprintf("[%s] ", loggerName), log.Ldate|log.Ltime|log.Lshortfile)
}

// PrintBigMessage 增强版大字符打印
func PrintBigMessage(message string) {
	const maxLines = 5
	patterns := loadCharacterPatterns()

	printSeparator(len(message))
	for line := 0; line < maxLines; line++ {
		fmt.Print("|  ")
		for _, char := range strings.ToUpper(message) {
			printLine(char, line, patterns)
		}
		fmt.Println("  |")
	}
	printSeparator(len(message))
}

func openLogFile(loggerName string) (*os.File, error) {
	filename := filepath.Join(logDir, fmt.Sprintf("%s.log", loggerName))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	return file, nil
}

func loadCharacterPatterns() map[rune][]string {
	return map[rune][]string{
		'A': {"  ███  ", " ████  ", "██  ██", "██████", "██  ██"},
		'B': {"█████ ", "██  ██", "█████ ", "██  ██", "█████ "},
		'C': {" █████", "██   ", "██   ", "██   ", " █████"},
		'D': {"████  ", "██  ██", "██  ██", "██  ██", "████  "},
		'E': {"█████", "██   ", "█████", "██   ", "█████"},
		'F': {"█████", "██   ", "█████", "██   ", "██   "},
		'G': {" █████", "██   ", "██ ███", "██  ██", " █████"},
		'H': {"██  ██", "██  ██", "██████", "██  ██", "██  ██"},
		'I': {"█████", "  ██  ", "  ██  ", "  ██  ", "█████"},
		'J': {"█████", "   ██ ", "   ██ ", "██ ██ ", " ███  "},
		'K': {"██  ██", "██ ██ ", "████  ", "██ ██ ", "██  ██"},
		'L': {"██   ", "██   ", "██   ", "██   ", "█████"},
		'M': {"██   ██", "███ ███", "██ █ ██", "██   ██", "██   ██"},
		'N': {"██   ██", "███  ██", "██ █ ██", "██  ███", "██   ██"},
		'O': {" ███ ", "██ ██", "██ ██", "██ ██", " ███ "},
		'P': {"█████ ", "██  ██", "█████ ", "██   ", "██   "},
		'Q': {" ███  ", "██ ██ ", "██ ██ ", "██ █  ", " ████ "},
		'R': {"█████ ", "██  ██", "█████ ", "██ ██ ", "██  ██"},
		'S': {" █████", "██   ", " ███ ", "   ██", "█████"},
		'T': {"█████", "  ██  ", "  ██  ", "  ██  ", "  ██  "},
		'U': {"██  ██", "██  ██", "██  ██", "██  ██", " ████ "},
		'V': {"██  ██", "██  ██", "██  ██", " ████ ", "  ██  "},
		'W': {"██   ██", "██   ██", "██ █ ██", "███ ███", "██   ██"},
		'X': {"██   ██", " ████ ", "  ██  ", " ████ ", "██   ██"},
		'Y': {"██   ██", " ████ ", "  ██  ", "  ██  ", "  ██  "},
		'Z': {"█████", "   ██", "  ██ ", " ██  ", "█████"},
	}
}

func printLine(char rune, line int, patterns map[rune][]string) {
	if pattern, exists := patterns[char]; exists && line < len(pattern) {
		fmt.Print(pattern[line])
	} else {
		fmt.Print(strings.Repeat(" ", 7))
	}
	fmt.Print("  ")
}

func printSeparator(length int) {
	fmt.Println("+" + strings.Repeat("-", length*9-1) + "+")
}
