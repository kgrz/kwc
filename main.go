package main

import (
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
	bufStates := make([]bufferState, totalRunsNeeded)
	for index := 0; index < totalRunsNeeded; index++ {
		// make a buffer of size 8192
		buf := make([]byte, BufferSize)
		// We get the offset based on the actual offset and bytes read in this iteration func.
		// TODO: we may have to read from the next byte. Need to check how offset works
		offset := ci.offset + int64(index*BufferSize)
		f.ReadAt(buf, offset)
		bufStates[index] = countBuffer(buf)
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

type bufferState struct {
	words, lines, chars int
	firstChar, lastChar byte
}

func isSpace(char byte) bool {
	return char == 32 || char == 9
}

func isNewLine(char byte) bool {
	return char == 10
}

// Implements the main character, word, line counting routines.
func countBuffer(buf []byte) bufferState {
	var bs bufferState
	bufSize := len(buf)
	bs.chars += utf8.RuneCount(buf)
	bs.lastChar = buf[bufSize-1]
	bs.firstChar = buf[0]

	var isPrevCharSpace bool

	// Special case for the first character. If it's a space, then set the
	// previous char pointer to true.
	bs.chars++
	if isSpace(bs.firstChar) || isNewLine(bs.firstChar) {
		isPrevCharSpace = true
	} else {
		isPrevCharSpace = false
	}

	if isNewLine(bs.firstChar) {
		bs.lines++
	}

	for i := 1; i < bufSize; i++ {
		// For each line, start from the second byte from the slice
		char := buf[i]
		if isNewLine(char) {
			bs.lines++
		}
		if isSpace(char) || isNewLine(char) {
			if !isPrevCharSpace {
				bs.words++
			}
			isPrevCharSpace = true
		} else {
			isPrevCharSpace = false
		}
	}

	// If the previous character (last of the line) is not a space, increment
	// the word bs, but only if the line has some characters.
	if !isPrevCharSpace {
		bs.words++
	}

	return bs
}
