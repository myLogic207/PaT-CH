package util

import (
	"errors"
	"os"
)

var (
	ErrDirNotWritable = errors.New("directory is not accessible")
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
