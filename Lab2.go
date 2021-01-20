package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Status struct {
	FileSize      int64
	TotalReceived int64
	Time          int
	Diffs         [5]int64
}
func NewStatus(fileLength int64) *Status {
	return &Status{FileSize: fileLength}
}
func (s *Status) Write(bytes []byte) (int, error) {
	n := len(bytes)
	s.TotalReceived += int64(n)
	s.Diffs[4] += int64(n)
	return n, nil
}
func (s *Status) PrintTable(fileName string)  {
	s.PrintHeader(fileName)
	for s.FileSize == -1 || s.FileSize > s.TotalReceived {
		s.PrintRow()
		s.Time++
		s.ShiftDiffs()
		time.Sleep(time.Second)
	}
}
func (s *Status) ShiftDiffs(){
	s.Diffs[0] = s.Diffs[1]; s.Diffs[1] = s.Diffs[2]; s.Diffs[2] = s.Diffs[3]; s.Diffs[3] = s.Diffs[4]; s.Diffs[4] = 0
}
func (s *Status) PrintHeader(fileName string) {
	fmt.Println("[Info] File name:", fileName)
	fmt.Println("  Time | Received | FileSize |    %    |    Speed    ")
	fmt.Println("=======|==========|==========|=========|=============")
}
func (s *Status) PrintRow() {
	var diffSum int64
	var fileSize string
	var percantage string
	for _, x := range s.Diffs {
		diffSum += x
	}
	if s.FileSize == -1 {
		fileSize = "--------"
		percantage = "-------"
	} else {
		fileSize = SizeToStr(s.FileSize)
		p := float64(s.TotalReceived) / float64(s.FileSize) * 100
		percantage = fmt.Sprintf("%0.2f%%", p)
	}
	speed := float64(diffSum) / 5 / 1024
	fmt.Printf(" %4ds | %8s | %8s | %7s | %7.2fKB/s\n",
		s.Time, SizeToStr(s.TotalReceived), fileSize, percantage, speed)
}
func GetFileName(link string) string {
	deniedSymbols := []string{ "\\", "/", ":", "*", "?", "\"", "<", ">", "|", "+", "!", "%", "@", }
	parts := strings.Split(link, "/")
	name := parts[len(parts) - 1]
	for _, x := range deniedSymbols {
		i := strings.LastIndex(name, x)
		if i != -1 {
			name = name[:i]
		}
	}
	return name
}
func SizeToStr(size int64) string {
	postfixes := []string{ "B", "KB", "MB", "GB", "TB"}
	fsize := float64(size)
	i := 0
	for ; fsize > 1024 && i < 4; i++ {
		fsize /= 1024
	}
	return fmt.Sprintf("%.2f%s", fsize, postfixes[i])
}
func main() {
	fmt.Print("Link: ")
	var link string
	fmt.Scan(&link)
	fileName := GetFileName(link)
	if fileName == "" {
		fileName = "TempName"
		fmt.Fprintf(os.Stderr,
			"[Worning] Failed to get a file name from the link so \"%s\" will be used \n",
			fileName)
	}
	response, err := http.Get(link)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %s \n", err.Error())
		os.Exit(1)
	}
	file, err := os.Create(fileName + ".tmp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %s \n", err.Error())
		os.Exit(2)
	}
	status := NewStatus(response.ContentLength)
	teeReader := io.TeeReader(response.Body, status)
	go status.PrintTable(fileName)
	if n, err := io.Copy(file, teeReader); err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %s \n", err.Error())
		os.Exit(3)
	} else {
		status.FileSize = n
	}
	status.PrintRow()
	if response.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr,
			"[Worning] The response status is \"%s\". The resulting file name is \"%s\" \n",
			response.Status, fileName +".tmp")
	} else {
		os.Rename(fileName + ".tmp", fileName)
		fmt.Printf("[Info] The file %s has been successfully downloaded\n", fileName)
	}
}
