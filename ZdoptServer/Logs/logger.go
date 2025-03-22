package Logs

import (
	"fmt"
	"os"
	"sync"
)

// ZLogger 结构体包含一个 logger 实例
type ZLogger struct {
	*Logger
	mu         sync.Mutex
	loggerName string
}

// NewZLogger 创建一个新的 ZLogger 实例
func NewZLogger(loggerName string, level Level) (*ZLogger, error) {
	logger, err := NewLogger(level, loggerName)
	if err != nil {
		return nil, fmt.Errorf("创建日志器失败: %w", err)
	}

	return &ZLogger{
		Logger:     logger,
		loggerName: loggerName,
	}, nil
}

// SetLevel 动态设置日志级别
func (zl *ZLogger) SetLevel(level Level) {
	zl.mu.Lock()
	defer zl.mu.Unlock()
	zl.level = level
}

// Log 线程安全日志记录
func (zl *ZLogger) Log(level Level, message string) {
	if level < zl.level {
		return
	}

	zl.mu.Lock()
	defer zl.mu.Unlock()

	prefix := fmt.Sprintf("[%s] ", level.String())
	zl.Logger.SetPrefix(prefix)
	zl.Logger.Println(message)
}

// Debug 调试日志
func (zl *ZLogger) Debug(message string) {
	zl.Log(Debug, message)
}

// Info 信息日志
func (zl *ZLogger) Info(message string) {
	zl.Log(Info, message)
}

// Warn 警告日志
func (zl *ZLogger) Warn(message string) {
	zl.Log(Warn, message)
}

// Error 错误日志
func (zl *ZLogger) Error(message string) {
	zl.Log(Error, message)
}

// Fatal 致命错误日志（带资源清理）
func (zl *ZLogger) Fatal(message string) {
	zl.mu.Lock()
	defer zl.mu.Unlock()

	zl.Logger.SetPrefix("[FATAL] ")
	zl.Logger.Println(message)

	// 关闭日志文件
	if zl.writer != nil {
		if closer, ok := zl.writer.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				fmt.Printf("关闭日志文件失败: %v\n", err)
			}
		}
	}

	os.Exit(1)
}

// Rotate 日志轮转（示例实现）
func (zl *ZLogger) Rotate() error {
	zl.mu.Lock()
	defer zl.mu.Unlock()

	if zl.writer != nil {
		if file, ok := zl.writer.(*os.File); ok {
			if err := file.Close(); err != nil {
				return fmt.Errorf("关闭旧日志文件失败: %w", err)
			}
		}
	}

	newFile, err := openLogFile(zl.loggerName)
	if err != nil {
		return fmt.Errorf("创建新日志文件失败: %w", err)
	}

	zl.Logger.SetOutput(newFile)
	zl.writer = newFile
	return nil
}
