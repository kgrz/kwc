package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"runtime/pprof"
	"text/tabwriter"
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
	return fmt.Sprintf("\t%d\t%d\t%d\t", c.lines, c.words, c.chars)
}

// Trying to align the byte count that's used to read the data
// to match the page cache of linux machines (4KB that I read from an article.
// Not sure if it's wrong, or optimal. I've seen at least 2 second decrease in
// net time after this change, so I'm keeping it.
const BufferSize = 4000 * 4000

func main() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
	defer w.Flush()

	cpuFile, err := os.Create("cpu.prof")
	handle(err)
	defer cpuFile.Close()
	defer pprof.StopCPUProfile()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		fmt.Println("could not start CPU profile: ", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		// Assume input to be stdin
		stream := bufio.NewScanner(os.Stdin)
		fmt.Println(processStream(stream))
	} else {
		// If the argument list is more than 1, assume that all the arguments
		// are file names. Verify if they are actually files, and run the
		// counting routine on each of the items
		filenames := os.Args[1:]
		validateFiles(filenames)
		counts := make([]Chunk, len(filenames))

		for i, filename := range filenames {
			counts[i] = countFile(filename)
			fmt.Fprintln(w, fmt.Sprintf("%s\t%s", counts[i], filename))
		}

		if len(filenames) > 1 {
			fmt.Fprintln(w, fmt.Sprintf("%s\ttotal", reduce(counts)))
		}
	}
}

func countFile(filename string) Chunk {
	f, err := os.Open(filename)
	handle(err)
	defer f.Close()
	chunks := fileOffsets(f)

	if len(chunks) == 1 {
		stream := bufio.NewScanner(f)
		return processStream(stream)
	}

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
	return finalState
}

// function for straightline reading. This is intended to work with
// streams.
func processStream(s *bufio.Scanner) Chunk {
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

func fileOffsets(f *os.File) []Chunk {
	fileinfo, err := f.Stat()
	handle(err)
	fileSize := fileinfo.Size()

	// If the file size is smaller than the buffer size, just create one chunk.
	// This avoids useless synchronisation cost. Even this is not optimal. This
	// will still use mutiple CPUs for a file with size 8193 bytes. Ideally,
	// this threshold value should be obtained by running it on a true
	// multicore machine on files of different sizes.
	if fileSize < BufferSize {
		return offset(fileSize)
	}

	return offsets(fileSize)
}

func offset(size int64) []Chunk {
	ci := make([]Chunk, 1)
	ci[0] = Chunk{size: size}
	return ci
}

func offsets(size int64) []Chunk {
	cpus := runtime.NumCPU()
	ci := make([]Chunk, cpus)

	chunkSize := size / int64(cpus)
	remainder := size % int64(cpus)
	var offset int64

	for i := 0; i < cpus; i++ {
		if i == cpus-1 {
			chunkSize = chunkSize + remainder
		}
		ci[i] = Chunk{size: chunkSize, offset: offset}
		offset += chunkSize
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

func validateFiles(filelist []string) {
	for _, file := range filelist {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("File does not exist: %s\n", file)
			fmt.Println("Aborting")
			os.Exit(1)
		}
	}	
}
