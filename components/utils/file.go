// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of L0
// 
// The L0 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// The L0 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// 
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// OpenFile opens or creates a file
// If the file already exists, open it . If it does not,
// It will create the file with mode 0644.
func OpenFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return f, err
}

// IsDirMissingOrEmpty determines whether a directory is empty or missing
func IsDirMissingOrEmpty(path string) (bool, error) {
	dirExists, err := IsDirExist(path)
	if err != nil {
		return false, err
	}

	if !dirExists {
		return true, nil
	}

	dirEmpty, err := IsDirEmpty(path)
	if err != nil {
		return false, err
	}

	if dirEmpty {
		return true, nil
	}

	return false, nil
}

// IsDirEmpty determines whether a directory is empty
func IsDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// IsDirExist determines whether a directory exists
func IsDirExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// AppDataDir returns a default data directory for the databases
func AppDataDir() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		if usr, err := user.Current(); err != nil {
			homeDir = usr.HomeDir
		}
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(homeDir, "AppData", "Roaming", "lcnd")
	default:
		return filepath.Join(homeDir, ".lcnd")
	}
}

// OpenDir opens or creates a dir
// If the dir already exists, open it . If it does not,
// It will create the file with mode 0700.
func OpenDir(dir string) (string, error) {
	exists, err := IsDirExist(dir)
	if !exists {
		err = os.MkdirAll(dir, 0700)
	}
	return dir, err
}

// FileExist determines whether a file exists
func FileExist(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}
