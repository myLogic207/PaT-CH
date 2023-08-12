package util

import (
	"errors"
	"math/rand"
	"os"
)

var (
	ErrDirNotWritable = errors.New("directory is not accessible")
)

const (
	RANDOM_LETTERS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890#@$^&*()_+{}[]"
)

func EnsureDir(path string) error {
	if fileInfo, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, 0777); err != nil {
				return err
			}
		}
	} else if !fileInfo.IsDir() || fileInfo.Mode().Perm() != 0777 {
		return ErrDirNotWritable
	}
	return nil
}

func GenerateRandomString(length int, letterString string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letterString[rand.Int63()%int64(len(letterString))]
	}
	return string(b)
}
