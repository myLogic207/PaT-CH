package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func CreateLogger(system_name string) (*log.Logger, error) {
	log.Println("preparing logger for (sub)system: ", system_name)

	writer_list := []io.Writer{}

	if env, ok := os.LookupEnv("ENVIRONMENT"); ok && strings.ToLower(env) == "development" {
		if val, ok := os.LookupEnv("DEV_NO_LOG"); !ok || strings.ToLower(val) != "true" {
			writer_list = append(writer_list, os.Stdout)
		}
	}

	file_writer, err := os.OpenFile("logs/"+system_name+".log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error while opening log file: %w", err)
	}

	writer_list = append(writer_list, file_writer)

	writer := io.MultiWriter(writer_list...)
	logger := log.New(writer, system_name+": ", log.Ltime|log.LstdFlags|log.Lshortfile)
	log.Println("logger prepared")
	return logger, nil
}
