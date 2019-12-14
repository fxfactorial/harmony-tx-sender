package utils

import (
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"
)

// FetchReceivers - fetch a list of proxies from a specified file
func FetchReceivers(filePath string) ([]string, error) {
	data, err := ReadFileToString(filePath)

	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	return lines, nil
}

// RandomReceiver - fetches a random proxy from the specified file
func RandomReceiver(receivers []string) string {
	rand.Seed(time.Now().Unix())
	receiver := receivers[rand.Intn(len(receivers))]

	return receiver
}

// ReadFileToString - check if a file exists, proceed to read it to memory if it does
func ReadFileToString(filePath string) (string, error) {
	if fileExists(filePath) {
		data, err := ioutil.ReadFile(filePath)

		if err != nil {
			return "", err
		}

		return string(data), nil
	} else {
		return "", nil
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
