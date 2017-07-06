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

func (c *chunk) String() string {
	return fmt.Sprintf(
		"offset: %d\nsize: %d\nwords: %d\nchars: %d\nlines: %d\nfirstByte: %d\nlastChar: %d\n",
		c.offset,
		c.size,
		c.words,
		c.chars,
		c.lines,
		c.firstByte,
		c.lastByte,
	)
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

		for i := 0; i < len(chunks); i++ {
			wg.Add(1)
			go func(index int) {
				chnkptr := &chunks[index]
				/*
					Note that above is not the same as:

					chnk := chunks[index]
					processBuffer(&chnk, f)

					Doing so does update that chunk object that is referenced
					by the variable chnk, but this won't update the value in
					the slice. So we'll need extra step of assigning the value
					back to the slice:

					chnk := chunks[index]
					processBuffer(&chnk, f)
					chunks[index] = chnk

					Thanks, Dave Cheney!
					https://dave.cheney.net/2017/04/29/there-is-no-pass-by-reference-in-go
				*/
				processBuffer(chnkptr, f)
				wg.Done()
			}(i)
		}

		wg.Wait()
		finalState = reduce(chunks)
	} else {
		processBuffer(&chunks[0], f)
		finalState = reduce(chunks)
	}

	fmt.Println("chars: ", finalState.chars)
	fmt.Println("words: ", finalState.words)
	fmt.Println("lines: ", finalState.lines)
}

func processBuffer(chunck *chunk, f *os.File) {
	leftOverBytes := int(chunck.size % BufferSize)

	runs := int(chunck.size / BufferSize)
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
		offset := chunck.offset + int64(index*BufferSize)
		if _, err := f.ReadAt(buf, offset); err != nil {
			log.Fatal(err)
		}

		chunck.chars += bufSize

		// setting up initial state for the first run of this chunk
		if index == 0 {
			firstByte := buf[0]
			chunck.firstByte = firstByte
			chunck.lastByte = firstByte
		}

		for _, char := range buf {
			if isNewLine(char) {
				chunck.lines++
			}

			// This also handles the case where the previous buffer's last char
			// was a space, and the first char in the current buffer is also a
			// space. In that case, since the lastByte and char are the same
			// values (we set it above), this if condition will be false, and
			// we won't be incrementing the count of words.
			if isSpace(char) && !isSpace(chunck.lastByte) {
				chunck.words++
			}
			chunck.lastByte = char
		}
	}
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

	// special handling for the last byte of the entire file. Increment word
	// count if last byte is not a space because that's the end of the line
	// (eof), but we don't actually get to read it.
	if lastChunk := chunks[chunksCount-1]; !isSpace(lastChunk.lastByte) {
		finalChunk.words++
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
