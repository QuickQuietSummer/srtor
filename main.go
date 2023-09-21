package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"log"
	"os"
	"path/filepath"
	"strings"
	"subtrans/pkg/fs"
	"subtrans/pkg/trans"
	"subtrans/pkg/util"
	"sync"
)

const targetDirName = "subtrans"

func main() {
	directory, err := askDirectory()
	if err != nil {
		panic(err)
	}

	files, err := fs.ScanDir(directory, func(f os.DirEntry) bool {
		return strings.HasSuffix(f.Name(), ".srt")
	})
	if err != nil {
		panic(err)
	}

	err = fs.MkdirOrIgnore(filepath.Join(directory, targetDirName))
	if err != nil {
		panic(err)
	}

	bar := progressbar.Default(int64(len(files)))

	wg := sync.WaitGroup{}
	goroutinesCount := 50
	chunks := util.ChunkSlice(files, len(files)/goroutinesCount)

	for i := range chunks {
		wg.Add(1)
		go func(chunk []string, bar *progressbar.ProgressBar) {
			defer wg.Done()
			err := processChunk(chunk, bar)
			if err != nil {
				log.Println(err)
			}
		}(chunks[i], bar)
	}
	wg.Wait()
	bye(len(files), directory)
}
func processChunk(chunk []string, bar *progressbar.ProgressBar) error {
	for _, f := range chunk {
		f := f
		err := processFile(f)
		if err != nil {
			return err
		}
		err = bar.Add(1)
		if err != nil {
			return err
		}
	}
	return nil
}
func bye(filesCount int, filesDir string) {
	fmt.Println()
	message := ""
	if filesCount > 0 {
		message = "Success"
	} else {
		abs, err := filepath.Abs(filesDir)
		if err != nil {
			log.Println(err)
		}
		message = fmt.Sprintf("No .srt files in %s", abs)
	}

	result := fmt.Sprintf("%s\n%s", message, "Press ENTER for exit")

	s := bufio.NewScanner(os.Stdin)
	fmt.Println(result)
	s.Scan()
}
func processFile(path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	target, err := trans.Translate(string(source), "en", "ru")
	if err != nil {
		return err
	}

	sourceDir := filepath.Dir(path)
	sourceName := filepath.Base(path)
	targetPath := filepath.Join(sourceDir, targetDirName, sourceName)
	targetBytes := []byte(target)
	unicodeReplacement := []byte{0xef, 0xbf, 0xbd}
	targetBytes = bytes.ToValidUTF8(targetBytes, unicodeReplacement)

	err = os.WriteFile(targetPath, targetBytes, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func askDirectory() (string, error) {
	s := bufio.NewScanner(os.Stdin)

	fmt.Println("Type directory absolute path which contains srt files")

	pathFromWd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("Or empty for default (%s)\n", pathFromWd)

	s.Scan()
	pathFromConsole, err := util.CanonizePath(s.Text())
	if err != nil {
		return "", err
	}
	result := ""
	if pathFromConsole != "" {
		result = pathFromConsole
	} else {
		result = pathFromWd
	}

	return result, nil
}
