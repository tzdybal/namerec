package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xiam/exif"
)

func listFiles(dir string) (fullPaths, files []string, err error) {
	dirEntries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			subPaths, subFiles, err := listFiles(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, nil, err
			}
			fullPaths = append(fullPaths, subPaths...)
			files = append(files, subFiles...)
		} else {
			fullPaths = append(fullPaths, filepath.Join(dir, entry.Name()))
			files = append(files, entry.Name())
		}
	}

	return
}

func getDateTime(tags map[string]string) (string, error) {
	keys := []string{"Date and Time", "Date and Time (Original)"} // more may be needed

	for _, k := range keys {
		if tags[k] != "" {
			return tags[k], nil
		}
	}

	return "", errors.New("date and time not found")
}

func isVideo(file string) bool {
	switch filepath.Ext(file) {
	case ".mp4":
		return true
	default:
		return false
	}
}

func isImage(file string) bool {
	switch filepath.Ext(file) {
	case ".jpg":
		return true
	default:
		return false
	}
}

func recoverVideo(dst, file string) error {
	// https://github.com/Martchus/tageditor
	output, err := exec.Command("tageditor", "-i", "-f", file).CombinedOutput()
	if err != nil {
		return err
	}

	reader := bytes.NewReader(output)
	scanner := bufio.NewScanner(reader)
	tag := "Creation time" // TODO: const

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, tag); idx != -1 {
			date := strings.TrimSpace(line[idx+len(tag):])
			copyAndTouch(dst, file, "VID", date)
			return nil
		}
	}

	return errors.New("metadata not found")
}

func recoverImage(dst, input string) error {
	data, err := exif.Read(input)
	if err != nil {
		return err
	}
	dateTime, err := getDateTime(data.Tags)
	if err != nil {
		return err
	}

	copyAndTouch(dst, input, "IMG", strings.Replace(dateTime, ":", "-", 2))

	return nil
}

func copyAndTouch(destination, input, prefix, dateTime string) error {
	var err error
	output := filepath.Clean(fmt.Sprintf("%s/%s_%s.jpg", destination, prefix, strings.Replace(strings.Replace(dateTime, ":", "", -1), " ", "_", -1)))

	fmt.Printf("cp %s -> %s\n", input, output)

	err = exec.Command("cp", input, output).Run()
	if err != nil {
		return fmt.Errorf("cp: %v", err)
	}

	err = exec.Command("touch", "-d", dateTime, output).Run()
	if err != nil {
		return fmt.Errorf("touch: %v", err)
	}

	return nil
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <src dir> <dst dir>\n", os.Args[0])
	}

	src := os.Args[1]
	dst := os.Args[2]

	fmt.Println("ensuring that", dst, "exists")
	os.MkdirAll(dst, os.ModePerm)

	srcPaths, _, err := listFiles(src)
	if err != nil {
		log.Fatal(err)
	}

	for _, input := range srcPaths {
		if isImage(input) {
			err = recoverImage(dst, input)
		} else if isVideo(input) {
			err = recoverVideo(dst, input)
		}
		if err != nil {
			log.Printf("%s: %v\n", input, err)
		}
	}
}
