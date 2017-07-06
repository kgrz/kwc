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

type chunk struct {
	offset, size        int64
	words, chars, lines int
	firstByte, lastByte byte
}

const BufferSize = 8192

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
	chunks := findOffsets(f, cpuCount)
	var finalState chunk

	// Fast loop if there is only one offset. This will be the case when the
	// file size is smaller than BufferSize. Check findOffsets for more details
	if len(chunks) > 1 {
		var wg sync.WaitGroup

		fmt.Printf("Using %d cores\n\n", cpuCount)
		newchunks := make([]chunk, cpuCount)

		for i, offset := range chunks {
			wg.Add(1)
			go func(idx int, off chunk) {
				newchunks[idx] = processBuffer(off, f)
				wg.Done()
			}(i, offset)
		}

		wg.Wait()
		finalState = reduce(newchunks)
	} else {
		finalState = processBuffer(chunks[0], f)
	}

	fmt.Println("chars: ", finalState.chars)
	fmt.Println("words: ", finalState.words)
	fmt.Println("lines: ", finalState.lines)
}

func processBuffer(ci chunk, f *os.File) chunk {
	var finalState chunk
	leftOverBytes := int(ci.size % BufferSize)

	runs := int(ci.size / BufferSize)
	if leftOverBytes > 0 {
		runs++
	}

	for index := 0; index < runs; index++ {
		// make a buffer of size 8192 or left over bytes depending
		bufSize := BufferSize
		if index == runs-1 && leftOverBytes > 0 {
			bufSize = leftOverBytes
		}
		buf := make([]byte, bufSize)
		// We get the offset based on the actual offset and bytes read in this
		// iteration func. TODO: we may have to read from the next byte. Need
		// to check how offset works
		offset := ci.offset + int64(index*BufferSize)
		_, err := f.ReadAt(buf, offset)
		if err != nil {
			log.Fatal(err)
		}
		countBuffer(buf, &finalState)
	}
	return finalState
}

func findOffsets(f *os.File, bufCount int) []chunk {
	fileinfo, err := f.Stat()
	handle(err)
	fileSize := fileinfo.Size()
	// If the file size is smaller than the buffer size, just create one chunk.
	// This avoids useless synchronisation cost. Even this is not optimal. This
	// will still use mutiple CPUs for a file with size 8193 bytes. Ideally,
	// this threshold value should be obtained by running it on a true
	// multicore machine on files of different sizes.
	if fileSize < BufferSize*BufferSize*10 {
		ci := make([]chunk, 1)
		ci[0] = chunk{size: fileSize}
		return ci
	}

	ci := make([]chunk, bufCount)

	size := fileSize / int64(bufCount)
	remainder := fileSize % int64(bufCount)
	var offset int64

	for i := 0; i < bufCount; i++ {
		if i == bufCount-1 {
			size = size + remainder
		}
		ci[i] = chunk{size: size, offset: offset}
		offset += size
	}

	return ci
}

func reduce(chunks []chunk) chunk {
	finalChunk := chunks[0]
	chunksCount := len(chunks)

	for i := 1; i < chunksCount; i++ {
		currentChunk := chunks[i]

		finalChunk.chars += currentChunk.chars
		finalChunk.lines += currentChunk.lines
		finalChunk.words += currentChunk.words
		finalChunk.lastByte = currentChunk.lastByte
	}

	return finalChunk
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
func countBuffer(buf []byte, chnk *chunk) {
	bufSize := len(buf)
	chnk.chars += bufSize

	for i := 0; i < bufSize; i++ {
		// For each line, start from the second byte from the slice
		char := buf[i]
		// relying on the assumption that a nul byte doesn't appear in mainline
		// operation. The initial value of a byte is 0.
		if chnk.firstByte == 0 {
			chnk.firstByte = char
			chnk.lastByte = char
		}

		if isNewLine(char) {
			chnk.lines++
		}

		if isSpace(char) {
			// This also handles the case where the previous buffer's last char
			// was a space, and the first char in the current buffer is also a
			// space. In that case, since the lastByte and char are the same
			// values (we set it above), this if condition will be false, and
			// we won't be incrementing the count of words.
			if !isSpace(chnk.lastByte) {
				chnk.words++
			}
		}
		chnk.lastByte = char
	}
}
