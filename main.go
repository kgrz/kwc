package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
)

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type chunkInfo struct {
	offset, size int64
}

func (ci chunkInfo) String() string {
	return fmt.Sprintf("offset: %d, size: %d", ci.offset, ci.size)
}

const BufferSize = 8192

var wg sync.WaitGroup

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Wrong number of arguments. Basic usage: go run main.go <filename>")
	}

	if len(os.Args) > 2 {
		fmt.Printf("Warning: Multile files are not supported yet. Using the first one.\n\n")
	}

	filename := os.Args[1]
	f, err := os.Open(filename)
	handle(err)
	defer f.Close()

	cpuCount := runtime.NumCPU()
	offsets := findOffsets(f, cpuCount)
	var finalState bufferState

	// Fast loop if there is only one offset. This will be the case when the file size is smaller than BufferSize. Check findOffsets for more details
	if len(offsets) > 1 {
		fmt.Printf("Using %d cores\n\n", cpuCount)
		bufStates := make(bufferStates, cpuCount)
		wg.Add(cpuCount)

		for i, offset := range offsets {
			off := offset
			index := i
			go func() {
				bufStates[index] = processBuffer(off, f, true)
			}()
		}

		wg.Wait()
		finalState = bufStates.reduce(bufferState{})
	} else {
		finalState = processBuffer(offsets[0], f, false)
	}

	fmt.Println("chars: ", finalState.chars)
	fmt.Println("words: ", finalState.words)
	fmt.Println("lines: ", finalState.lines)
}

func processBuffer(ci chunkInfo, f *os.File, syncNeeded bool) bufferState {
	// This is fugly. The synchronisation should be done on the type, rather
	// than as a hacky flag.
	if syncNeeded {
		defer wg.Done()
	}
	runs := int(ci.size / BufferSize)
	leftOverBytes := int(ci.size % BufferSize)
	var buffers int
	if leftOverBytes > 0 {
		buffers = runs + 1
	} else {
		buffers = runs
	}
	bufStates := make(bufferStates, buffers)
	for index := 0; index < runs; index++ {
		// make a buffer of size 8192
		buf := make([]byte, BufferSize)
		// We get the offset based on the actual offset and bytes read in this
		// iteration func. TODO: we may have to read from the next byte. Need
		// to check how offset works
		offset := ci.offset + int64(index*BufferSize)
		_, err := f.ReadAt(buf, offset)
		if err != nil {
			log.Fatal(err)
		}
		bufStates[index] = countBuffer(buf)
	}
	// TODO: Fold this into the loop above
	if leftOverBytes > 0 {
		buf := make([]byte, leftOverBytes)
		// We get the offset based on the actual offset and bytes read in this
		// iteration func. TODO: we may have to read from the next byte. Need
		// to check how offset works
		offset := ci.offset + int64(runs*BufferSize)
		_, err := f.ReadAt(buf, offset)
		if err != nil {
			log.Fatal(err)
		}
		bufStates[buffers-1] = countBuffer(buf)
	}
	finalState := bufStates.reduce(bufferState{})
	return finalState
}

func findOffsets(f *os.File, bufCount int) []chunkInfo {
	fileinfo, err := f.Stat()
	handle(err)
	fileSize := fileinfo.Size()
	// If the file size is smaller than the buffer size, just create one chunk.
	// This avoids useless synchronisation cost. Even this is not optimal. This
	// will still use mutiple CPUs for a file with size 8193 bytes. Ideally,
	// this threshold value should be obtained by running it on a true
	// multicore machine on files of different sizes.
	if fileSize < BufferSize {
		ci := make([]chunkInfo, 1)
		ci[0] = chunkInfo{size: fileSize, offset: 0}
		return ci
	}

	ci := make([]chunkInfo, bufCount)

	size := fileSize / int64(bufCount)
	remainder := fileSize % int64(bufCount)
	var offset int64

	for i := 0; i < bufCount; i++ {
		if i == bufCount-1 {
			size = size + remainder
		}
		ci[i] = chunkInfo{size: size, offset: offset}
		offset += size
	}

	return ci
}

type bufferState struct {
	words, lines, chars int
	firstChar, lastChar byte
}

type bufferStates []bufferState

func (bxs bufferStates) reduce(initialState bufferState) bufferState {
	var previousState bufferState
	finalState := initialState
	bxsCount := len(bxs)

	for i, bufState := range bxs {
		if i == 0 {
			previousState = initialState
			finalState.firstChar = bufState.firstChar
		} else {
			previousState = bxs[i-1]
			// special treatment for words because we need to check with previous
			// state if past buffer cut off some word partially. If the last char
			// in previous buffer and the first char in current buffer are not
			// spaces, we have counted two words, so reduce it by one
			if !isSpace(previousState.lastChar) && !isSpace(bufState.firstChar) {
				finalState.words--
			}
		}

		if i == bxsCount-1 {
			finalState.lastChar = bufState.lastChar
		}

		finalState.chars += bufState.chars
		finalState.lines += bufState.lines
		finalState.words += bufState.words

	}

	return finalState
}

func isSpace(char byte) bool {
	// According to manpages, a space is one of:
	// space
	// horizontal tab
	// vertical tab
	// carriage return
	// new line
	// form feed
	// NEL
	// NBSP
	return char == 32 || char == 9 || char == 11 || char == 13 || char == 10 || char == 12 || char == 0x85 || char == 0xA0
}

func isNewLine(char byte) bool {
	return char == 10
}

// Implements the main character, word, line counting routines.
func countBuffer(buf []byte) bufferState {
	var bs bufferState
	bufSize := len(buf)
	bs.chars += bufSize
	bs.lastChar = buf[bufSize-1]
	bs.firstChar = buf[0]

	var isPrevCharSpace bool

	// Special case for the first character. If it's a space, then set the
	// previous char pointer to true.
	if isSpace(bs.firstChar) {
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
		if isSpace(char) {
			if !isPrevCharSpace {
				bs.words++
			}
			isPrevCharSpace = true
		} else {
			isPrevCharSpace = false
		}
	}

	// If the previous character (last of the buffer) is not a space, increment
	// the word count
	if !isPrevCharSpace {
		bs.words++
	}

	return bs
}
