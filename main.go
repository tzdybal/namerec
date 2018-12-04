package main

import (
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
		data, err := exif.Read(input)
		if err != nil {
			if err != exif.ErrNoExifData {
				log.Print(err)
			}
			continue
		}
		dateTime, err := getDateTime(data.Tags)
		if err != nil {
			log.Print(err)
			continue
		}

		output := filepath.Clean(fmt.Sprintf("%s/IMG_%s.jpg", dst, strings.Replace(strings.Replace(dateTime, ":", "", -1), " ", "_", -1)))

		fmt.Printf("cp %s -> %s\n", input, output)

		err = exec.Command("cp", input, output).Run()
		if err != nil {
			log.Println("cp:", err)
		}

		err = exec.Command("touch", "-d", strings.Replace(dateTime, ":", "-", 2), output).Run()
		if err != nil {
			log.Println("touch:", err)
		}
	}
}
