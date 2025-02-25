package misc

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var (
	ErrLogger  = log.New(os.Stderr, "[ FAIL ]: ", log.Lshortfile)
	WarnLogger = log.New(os.Stderr, "[ WARN ]: ", log.Lshortfile)
	InfoLogger = log.New(os.Stdout, "[ INFO ]: ", log.Lshortfile)
)

func GetScriptName() string {
	_, scriptName := filepath.Split(os.Args[0])
	if _, scriptFile, _, ok := runtime.Caller(1); ok {
		_, scriptName = filepath.Split(scriptFile)
	}

	return scriptName
}

func CheckFileExists(path string) (bool, error) {
	// check if file exists
	info, err := os.Stat(path)

	if err == nil { // file exists
		mode := info.Mode()
		if !mode.IsRegular() {
			return false, fmt.Errorf("%s is not a regular file", path)
		}

		return true, nil
	} else if errors.Is(err, os.ErrNotExist) { // file does not exists
		return false, nil
	} else { // unable to check if file exists or not
		return false, err
	}
}

// Checks if executables exists.
//
// But it has some caveat.
//
// Let's say you are looking for hello.exe.
// But hello.exe is not in a path but rather in a same directory where your program runs.
//
// This function would say it couldn't find hello.exe
// since it's a relative path.
func CheckExeExists(exe string) bool {
	_, err := exec.LookPath(exe)
	return err == nil
}

func CopyFile(src, dst string, perm os.FileMode) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	var err error
	var srcFile []byte
	if srcFile, err = os.ReadFile(src); err != nil {
		return err
	}

	if err = os.WriteFile(dst, srcFile, perm); err != nil {
		return err
	}

	return nil
}

// permission is set to 0755
func MkDir(path string) error {
	path = filepath.Clean(path)

	InfoLogger.Printf("creating %s folder", path)

	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	return nil
}

func IsSamePath(pathA, pathB string) (bool, error) {
	absPathA, err := filepath.Abs(pathA)
	if err != nil {
		return false, err
	}
	absPathB, err := filepath.Abs(pathB)
	if err != nil {
		return false, err
	}

	return absPathA == absPathB, nil
}

func IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
