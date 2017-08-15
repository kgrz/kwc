package main

import (
	"bufio"
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

// This holds the state of one chunk of the file
type Chunk struct {
	offset, size        int64
	words, chars, lines int
	firstByte, lastByte byte
}

func (c Chunk) String() string {
	return fmt.Sprintf(
		"chars: %d\nwords: %d\nlines: %d\n",
		c.chars,
		c.words,
		c.lines,
	)
}

// Trying to align the byte count that's used to read the data
// to match the page cache of linux machines (4KB that I read from an article.
// Not sure if it's wrong, or optimal. I've seen at least 2 second decrease in
// net time after this change, so I'm keeping it.
const BufferSize = 4000 * 4000

func main() {
	useSingleChunk := false

	if len(os.Args) < 2 {
		useSingleChunk = true
	}

	if len(os.Args) == 2 {
		filename := os.Args[1]
		fileinfo, err := os.Stat(filename)
		handle(err)
		fileSize := fileinfo.Size()
		// If the file size is smaller than the buffer size, just create one chunk.
		// This avoids useless synchronisation cost. Even this is not optimal. This
		// will still use mutiple CPUs for a file with size 8193 bytes. Ideally,
		// this threshold value should be obtained by running it on a true
		// multicore machine on files of different sizes.
		if fileSize < BufferSize {
			useSingleChunk = true
		}
	}

	if useSingleChunk {
		stream := bufio.NewScanner(os.Stdin)

		if len(os.Args) == 2 {
			f, err := os.Open(os.Args[1])
			handle(err)
			defer f.Close()
			stream = bufio.NewScanner(f)
		}

		fmt.Println(process1(stream))
		os.Exit(0)
	}

	if len(os.Args) > 2 {
		log.Fatal("Error: Multile files are not supported yet.")
	}

	f, err := os.Open(os.Args[1])
	handle(err)
	defer f.Close()

	chunks := findOffsets(f)

	var wg sync.WaitGroup

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
	finalState := reduce(chunks)
	fmt.Println(finalState)
}

// function for straightline reading. This is intended to work with
// streams.
func process1(s *bufio.Scanner) Chunk {
	var chunk Chunk
	s.Split(bufio.ScanBytes)

	for s.Scan() {
		bytes := s.Bytes()
		chunk.chars += len(bytes)

		for _, char := range bytes {
			if isNewLine(char) {
				chunk.lines++
			}

			if isSpace(char) && (chunk.lastByte > 0 && !isSpace(chunk.lastByte)) {
				chunk.words++
			}

			chunk.lastByte = char
		}
	}

	return chunk
}

// function for buffered, chunked reading
func processBuffer(chunk *Chunk, f *os.File) {
	leftOverBytes := int(chunk.size % BufferSize)

	runs := int(chunk.size / BufferSize)
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
		// iteration func.
		offset := chunk.offset + int64(index*BufferSize)
		if _, err := f.ReadAt(buf, offset); err != nil {
			log.Fatal(err)
		}

		chunk.chars += bufSize

		// setting up initial state for the first run of this chunk
		if index == 0 {
			firstByte := buf[0]
			chunk.firstByte = firstByte
			chunk.lastByte = firstByte
		}

		for _, char := range buf {
			if isNewLine(char) {
				chunk.lines++
			}

			// This also handles the case where the previous buffer's last char
			// was a space, and the first char in the current buffer is also a
			// space. In that case, since the lastByte and char are the same
			// values (we set it above), this if condition will be false, and
			// we won't be incrementing the count of words.
			if isSpace(char) && !isSpace(chunk.lastByte) {
				chunk.words++
			}
			chunk.lastByte = char
		}
	}
}

func findOffsets(f *os.File) []Chunk {
	fileinfo, err := f.Stat()
	handle(err)
	fileSize := fileinfo.Size()
	cpus := runtime.NumCPU()

	ci := make([]Chunk, cpus)

	size := fileSize / int64(cpus)
	remainder := fileSize % int64(cpus)
	var offset int64

	for i := 0; i < cpus; i++ {
		if i == cpus-1 {
			size = size + remainder
		}
		ci[i] = Chunk{size: size, offset: offset}
		offset += size
	}

	fmt.Printf("Using %d cores\n\n", cpus)

	return ci
}

func reduce(chunk []Chunk) Chunk {
	finalChunk := chunk[0]
	chunksCount := len(chunk)

	for i := 1; i < chunksCount; i++ {
		currentChunk := chunk[i]

		finalChunk.chars += currentChunk.chars
		finalChunk.lines += currentChunk.lines
		finalChunk.words += currentChunk.words
		finalChunk.lastByte = currentChunk.lastByte
	}

	// special handling for the last byte of the entire file. Increment word
	// count if last byte is not a space because that's the end of the line
	// (eof), but we don't actually get to read it.
	if lastChunk := chunk[chunksCount-1]; !isSpace(lastChunk.lastByte) {
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
