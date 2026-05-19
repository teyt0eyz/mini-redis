package persistence

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

var (
	file *os.File
	mu   sync.Mutex
)

func Open(path string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	file = f
	return nil
}

func Append(cmd string) {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		fmt.Fprintln(file, cmd)
	}
}

func Replay(path string, exec func(string) string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			exec(line)
		}
	}
	return scanner.Err()
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		file.Close()
		file = nil
	}
}
