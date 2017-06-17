package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"unicode/utf8"
)

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type chunkInfo struct {
	offset, size int64
}

const BufferSize = 8192

func main() {
	f, err := os.Open("out.txt")
	handle(err)
	offsets := findOffsets(f)
	for _, offset := range offsets {
		processBuffer(offset, f)
	}
}

func processBuffer(ci chunkInfo, f *os.File) {
	totalRunsNeeded := int(ci.size / BufferSize)
	for index := 0; index < totalRunsNeeded; index++ {
		// make a buffer of size 8192
		buf := make([]byte, BufferSize)
		// We get the offset based on the actual offset and bytes read in this iteration func.
		// TODO: we may have to read from the next byte. Need to check how offset works
		offset := ci.offset + int64(index*BufferSize)
		f.ReadAt(buf, offset)
		counts := countBuffer(buf)
		fmt.Println(counts)
	}
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

// Counts contains the snapshot of the Word, Line, Char counts after the file is
// processed.
type Counts struct {
	Words int
	Lines int
	Chars int
}

func (c Counts) String() string {
	return fmt.Sprintf("line count: %d\nword count: %d\nchar count: %d", c.Lines, c.Words, c.Chars)
}

func isSpace(char byte) bool {
	return char == 32 || char == 9
}

func isNewLine(char byte) bool {
	return char == 10
}

// Implements the main character, word, line counting routines.
func countBuffer(buf []byte) Counts {
	var count Counts
	bufSize := len(buf)
	count.Chars += utf8.RuneCount(buf)

	var isPrevCharSpace bool

	// Special case for the first character. If it's a space, then set the
	// previous char pointer to true.
	count.Chars++
	if isSpace(buf[0]) || isNewLine(buf[0]) {
		isPrevCharSpace = true
	} else {
		isPrevCharSpace = false
	}

	if isNewLine(buf[0]) {
		count.Lines++
	}

	for i := 1; i < bufSize; i++ {
		// For each line, start from the second byte from the slice
		char := buf[i]
		if isNewLine(char) {
			count.Lines++
		}
		if isSpace(char) || isNewLine(char) {
			if !isPrevCharSpace {
				count.Words++
			}
			isPrevCharSpace = true
		} else {
			isPrevCharSpace = false
		}
	}

	// all the bytes until the last one on a line have been counted. If the
	// previous character (last of the line) is not a space, increment the word
	// count, but only if the line has some characters.
	if !isPrevCharSpace {
		count.Words++
	}

	return count
}
