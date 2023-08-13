package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const TIMEOUT = 5

var (
	LOGFOLDER, LOGFOLDEROLD, SUFFIX, REPLACECHAR string
	LOGFLAGS                                     int = 0
	logFileList                                      = []*os.File{}
	defaultLogConfig                                 = map[string]interface{}{
		"flags":       "date,time,microseconds,utc,shortfile,msgprefix",
		"suffix":      ".log",
		"folder":      "/var/log",
		"replacechar": "-",
	}
)

func SetDefaultLogger(config *Config) {
	config.MergeDefault(defaultLogConfig)
	if flags, ok := config.Get("flags").(string); ok {
		setDefaultLoggerFlags(flags)
	} else {
		log.Println("no default flags specified")
	}
	writerList := []io.Writer{}
	if env, ok := os.LookupEnv("ENVIRONMENT"); ok && strings.ToLower(env) == "development" {
		if val, ok := os.LookupEnv("DEV_NO_LOG"); !ok || strings.ToLower(val) != "true" {
			writerList = append(writerList, os.Stdout)
		}
	}
	if folderPath, ok := config.Get("folder").(string); ok {
		if err := EnsureDir(folderPath); err != nil {
			log.Println("error while creating log folder:", err)
		} else {
			LOGFOLDER = folderPath
		}
	} else {
		log.Println("no log folder specified")
	}

	if folderOldPath, ok := config.Get("folderold").(string); ok {
		if err := EnsureDir(folderOldPath); err != nil {
			log.Println("error while creating log folder:", err)
		} else {
			LOGFOLDEROLD = folderOldPath
		}
	} else {
		log.Println("no old log folder specified")
	}

	if useDefaultFile := config.GetBool("defaultfile"); useDefaultFile {
		filePath := filepath.Join(LOGFOLDER, "system")
		logFile, err := os.OpenFile(filePath+SUFFIX, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("error while opening log file:", err)
		} else {
			writerList = append(writerList, logFile)
			logFileList = append(logFileList, logFile)
		}
	} else {
		log.Println("no default log file specified")
	}
	log.SetOutput(io.MultiWriter(writerList...))

	if prefix, ok := config.Get("prefix").(string); ok {
		log.SetPrefix(prefix + " ")
	} else {
		log.Println("no default prefix specified")
	}

	if suffix, ok := config.Get("suffix").(string); ok {
		SUFFIX = suffix
	} else {
		log.Println("no default suffix specified")
	}

	if replaceChar, ok := config.Get("replacechar").(string); ok {
		REPLACECHAR = replaceChar
	} else {
		log.Println("no default replacechar specified")
	}
}

func setDefaultLoggerFlags(flags string) {
	if flags == "" {
		return
	}
	flagList := strings.Split(flags, ",")
	flagBuffer := 0
	for _, flag := range flagList {
		switch strings.ToLower(flag) {
		case "date":
			flagBuffer |= log.Ldate
		case "time":
			flagBuffer |= log.Ltime
		case "microseconds":
			flagBuffer |= log.Lmicroseconds
		case "utc":
			flagBuffer |= log.LUTC
		case "shortfile":
			flagBuffer |= log.Lshortfile
		case "longfile":
			flagBuffer |= log.Llongfile
		case "msgprefix":
			flagBuffer |= log.Lmsgprefix
		case "stdflags":
			flagBuffer |= log.LstdFlags
		default:
			log.Println("unknown flag", flag)
		}
	}
	LOGFLAGS = flagBuffer
	log.SetFlags(flagBuffer)
}

func CreateLogger(system_name string) (*log.Logger, error) {
	log.Println("preparing logger for (sub)system:", system_name)

	writerList := []io.Writer{}

	if env, ok := os.LookupEnv("ENVIRONMENT"); ok && strings.ToLower(env) == "development" {
		if val, ok := os.LookupEnv("DEV_NO_LOG"); !ok || strings.ToLower(val) != "true" {
			writerList = append(writerList, os.Stdout)
		}
	}

	if LOGFOLDER != "" {
		filePath := filepath.Join(LOGFOLDER, strings.ReplaceAll(system_name, " ", REPLACECHAR))
		logFile, err := os.OpenFile(filePath+SUFFIX, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("error while opening log file: %w", err)
		}
		writerList = append(writerList, logFile)
		logFileList = append(logFileList, logFile)
	} else {
		log.Println("no log folder specified")
	}

	writer := io.MultiWriter(writerList...)
	logFlags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.Lmsgprefix
	if LOGFLAGS != 0 {
		logFlags = LOGFLAGS
	}
	logger := log.New(writer, system_name+": ", logFlags)
	log.Println("logger prepared")
	return logger, nil
}

func TerminateLoggers() {
	finishChannel := make(chan bool, 1)
	log.Println("terminating loggers")
	go terminationWrapper(finishChannel)
	select {
	case <-finishChannel:
		break
	case <-time.After(TIMEOUT * time.Second):
		log.Println("timeout while waiting for loggers to terminate")
	}

}

func terminationWrapper(finishChannel chan bool) {
	waitGroup := sync.WaitGroup{}
	for _, logFile := range logFileList {
		waitGroup.Add(1)
		errorChan := make(chan error, 1)
		go terminateLogger(logFile, errorChan, &waitGroup)
	}
	waitGroup.Wait()
	finishChannel <- true
}

func terminateLogger(logFile *os.File, errorChan chan error, wg *sync.WaitGroup) {
	timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	log.Println("closing log file:", logFile.Name())
	logFile.Write([]byte(fmt.Sprintf("Log closed at %s\n", timestamp)))
	logFile.Sync()
	logFile.Close()
	// set Timestamp
	newName := fmt.Sprintf("%s.%s%s", strings.TrimSuffix(filepath.Base(logFile.Name()), SUFFIX), timestamp, SUFFIX)
	folder := LOGFOLDER
	if LOGFOLDEROLD != "" {
		folder = LOGFOLDEROLD
	}
	newName = filepath.Join(folder, newName)
	if err := os.Rename(logFile.Name(), newName); err != nil {
		log.Println("error while renaming log file:", logFile.Name())
		errorChan <- err
	}
	wg.Done()
}
