package util

import (
	"fmt"
	"log"
	"os"
	"time"
)

const debugLogPath = "/opt/go_cmdb/var/debug/web_release_debug.log"

// DebugLog 写入 WEB_RELEASE_DEBUG 日志（同时写 stdout 和文件）
func DebugLog(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	
	// 写入 stdout（通过 log）
	log.Print(msg)
	
	// 写入文件
	debugFile, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open debug log: %v", err)
		return
	}
	defer debugFile.Close()
	
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	debugFile.WriteString(fmt.Sprintf("%s %s\n", timestamp, msg))
}
