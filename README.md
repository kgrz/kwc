kwc
====


An attempt at file offset-based `wc` implementation that can use
multiple cores to read the same file. Outline:

1. Get number of CPUs on the machine
1. Create those many number of go routines that start reading the file
in chunks.
1. Offsets are created based on that number so that each chunk is read
starting from that offset.

__Mostly works on *nix machines__

Warning: Code in here is crap, don't read it.

Installation:
-------------

I haven't created releases or per-OS packages, so the only way to try
this out is via `go get`, which means you need to have working Go
installation.

```
go get github.com/kgrz/kwc
```

That _should_ compile and install the binary into your `$GOPATH`. Then run
the binary as `kwc`. If it's not there, then `cd` into
`$GOPATH/src/github.com/kgrz/kwc` and run `go install`.


Some problems:
--------------

1. I'm finding it non straight forward to do UTF-8 aware reading because
   if a chunk cuts an particular multi-byte character in the middle,
   that shouldn't be counted as two separate words! If we use
   `utf8.RuneCount()` on a slice that has a partial multi-byte word,
   that count can end up being wrong.

   Update: I think I have a solution for this! Will implement it soon.


What's the advantage?:
----------------------

It's fast™

<img src='https://user-images.githubusercontent.com/400299/27813278-6f917238-6092-11e7-9472-ac9125fa3ba8.gif' title='benchmarking wc vs this program'/>


Learnings:
---------

1. The `os.readAt` Go function internally uses the `pread` syscall which
   works well with multi-threaded access of the same file:
   http://man7.org/linux/man-pages/man2/pread.2.html

2. The initial implementation used a naive `isspace` function I wrote
   that only catered to spaces and tabs (ascii 32 and 9). But as per the
   man page of `wc` and `isspace` function that gets used in it, a
   "space" for the purposes of `wc` contains both a whitespace
   characters and new lines or equivalents:

    * ascii space (32)
    * ascii tab (9) \t
    * new line (10) \n
    * vertical tab (11) \v
    * form feed (12) \f
    * carriage return (13) \r
    * non breaking space (0xA0)
    * next line character (0x85)

3. Avoiding [`bufio.Scan()`](https://golang.org/pkg/bufio/#Scanner.Scan)
   is maybe something you'd want to consider if you're looking for
   speed. The `Scan()` function does a lot of things extra like basic
   consistent error handling, and it's very useful if you want to store
   the scanned bytes into lines/words for every iteration. We don't need
   to do that when just counting the characters or words, so we avoid
   using it. Perf impact is considerable.


   To do a basic test of this hypothesis, try running the program on a
   `cat`-ed output which uses the scanner codepath and compare it with
   `wc`.
