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
	firstChar, lastChar byte

	count
}

type count struct {
	wordct, linect, charct int
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
	chunks := getChunks(f, cpuCount)
	updatedChunks := make([]chunk, len(chunks))

	fmt.Printf("Using %d cores\n\n", cpuCount)
	wg.Add(len(chunks))

	for i, c := range chunks {
		a := c
		go func(updatedChunks []chunk, a chunk, index int) {
			updatedChunks[index] = processBuffer(&a, f)
		}(updatedChunks, a, i)
	}

	wg.Wait()
	for _, c := range updatedChunks {
		fmt.Println(c)
	}
	count := reduce(updatedChunks)

	fmt.Println("chars: ", count.charct)
	fmt.Println("words: ", count.wordct)
	fmt.Println("lines: ", count.linect)
}

func processBuffer(c *chunk, f *os.File) chunk {
	defer wg.Done()
	runs := int(c.size / BufferSize)
	leftOverBytes := int(c.size % BufferSize)

	if leftOverBytes > 0 {
		runs++
	}

	for index := 0; index < runs; index++ {
		// make a buffer of size 8192 (default) or left over bytes (if needed)
		var buf []byte
		if index == runs-1 && leftOverBytes > 0 {
			buf = make([]byte, leftOverBytes)
		} else {
			buf = make([]byte, BufferSize)
		}

		// We get the offset based on the actual offset and bytes read in this
		// iteration func. TODO: we may have to read from the next byte. Need
		// to check how offset works
		offset := c.offset + int64(index*BufferSize)
		_, err := f.ReadAt(buf, offset)
		if err != nil {
			log.Fatal(err)
		}
		countBuffer(c, buf)
	}

	return *c
}

func getChunks(f *os.File, bufCount int) []chunk {
	fileinfo, err := f.Stat()
	handle(err)
	fileSize := fileinfo.Size()
	// If the file size is smaller than the buffer size, just create one chunk.
	// This avoids useless synchronisation cost. Even this is not optimal. This
	// will still use mutiple CPUs for a file with size 8193 bytes. Ideally,
	// this threshold value should be obtained by running it on a true
	// multicore machine on files of different sizes.
	if fileSize < BufferSize {
		ci := make([]chunk, 1)
		ci[0] = chunk{size: fileSize, offset: 0}
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

func reduce(chunks []chunk) count {
	count := chunks[0].count

	for index := 1; index < len(chunks); index++ {
		prevChunk := chunks[index-1]
		currentChunk := chunks[index]

		count.charct += currentChunk.charct
		count.wordct += currentChunk.wordct
		count.linect += currentChunk.linect

		// special treatment for words because we need to check with previous
		// state if past buffer cut off some word partially. If the last char
		// in previous buffer and the first char in current buffer are not
		// spaces, we have counted two words, so reduce it by one
		if !isSpace(prevChunk.lastChar) && !isSpace(currentChunk.firstChar) {
			count.wordct--
		}
	}

	return count
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
func countBuffer(c *chunk, buf []byte) {
	bufSize := len(buf)
	c.charct += bufSize

	var isPrevCharSpace bool

	// Special case for the first character. If it's a space, then set the
	// previous char pointer to true.
	if isSpace(c.lastChar) {
		isPrevCharSpace = true
	}

	for i := 0; i < bufSize; i++ {
		// For each line, start from the second byte from the slice
		char := buf[i]
		// default value of byte is 0
		if c.firstChar == 0 {
			c.firstChar = char
		}

		if isNewLine(char) {
			c.linect++
		}
		if isSpace(char) {
			if !isPrevCharSpace {
				c.wordct++
			}
			isPrevCharSpace = true
		} else {
			isPrevCharSpace = false
		}
	}

	// If the previous character (last of the buffer) is not a space, increment
	// the word count
	if !isPrevCharSpace {
		c.wordct++
	}
	// set the last char of this buffer as the last char of the chunk. This gets changed for every loop and buffer.
	c.lastChar = buf[bufSize-1]
}
