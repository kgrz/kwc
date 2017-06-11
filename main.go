package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type chunkInfo struct {
	offset, size int64
}

func main() {
	f, err := os.Open("out.txt")
	handle(err)
	offsets := findOffsets(f)
	chunks := make([][]byte, len(offsets))
	for i, offset := range offsets {
		buf := make([]byte, offset.size)
		f.ReadAt(buf, offset.offset)
		chunks[i] = buf
	}
	fmt.Println(chunks)
}

func findOffsets(f *os.File) []chunkInfo {
	fileinfo, err := f.Stat()
	handle(err)
	fileSize := fileinfo.Size()
	bufCount := runtime.NumCPU()
	ci := make([]chunkInfo, bufCount)

	size := fileSize / int64(bufCount)
	remainder := fileSize % int64(bufCount)

	for i := 0; i < bufCount; i++ {
		if i == bufCount-1 {
			size = size + remainder
		}
		ci[i] = chunkInfo{size: size, offset: size * int64(i)}
	}

	return ci
}
