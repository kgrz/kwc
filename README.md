wcc
====

Warning: Code in here is crap, don't read it.

A borken attempt at file offset-based `wc` implementation. Outline of
the attempt is:

1. Get number of CPUs on the machine
1. Create those many number of go routines that start reading the file
in chunks.
1. Offsets are created based on that number so that each chunk is read
starting from that offset.


Couple of problems:
------------------

1. I'm pretty sure I'm doing the offset calculation part wrong. Will
   revisit it later.
2. I'm finding it hard to do UTF-8 aware reading because if a chunk cuts
   an particular multi-byte character in the middle, that shouldn't be
   counted as two separate words! If we use `utf8.RuneCount()` on a
   slice that has a partial multi-byte word, that count can end up being
   wrong. I don't yet have a solution for this.


But the speed seems amazing! I ran this program, and the GNU `wc`
program, and https://github.com/kgrz/wc program on a 16 core machine on
a 1.9GB file, and this is the result:

<img width="620" alt="screenshot 2017-06-18 08 25 52" src="https://user-images.githubusercontent.com/400299/27258671-6793ccbe-541e-11e7-92e7-1c49d7fbe366.png">


Wrong result, but hey, it's fast™ 🤷🏽‍♂️

Bonus bug! Current implementation has a race condition because the go rountine inside main() returns immediately, and the reduce function operates on the `bufStates` slice! 

Learnings:
---------

1. The readAt Go function internally uses the `pread` syscall which
works well with multi-threaded access of the same file: http://man7.org/linux/man-pages/man2/pread.2.html
