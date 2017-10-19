package main

import (
	"bufio"
	"strings"
	"testing"
)

type fixtures struct {
	input string
	words int
	lines int
	chars int
}

// Each of the test case results below are based on the output of `wc` program
// on a plain text file with the string contents. I'm trading off actual
// correctness in favour of verifiable comparision between wc and my wc program.
// For example, one would expect a hyphenated word to count as one, but the
// *nix wc program counts them as two words.
var tests = []fixtures{
	{"this is a test string", 5, 0, 21},
	{"this is a test string ", 5, 0, 22},
	{" this is a test string", 5, 0, 22},
	{"this   is          a     test string", 5, 0, 36},
	{"this	is			a		test string", 5, 0, 24},
	{"this is a test string\nthis is another strirg", 9, 1, 44},
	{"this is a test string\n\n\n\n\n\nthis is another string", 9, 6, 49},
	{"this is a test string\n this is another string", 9, 1, 45},
	{"this is a test str-\ning this is another string", 10, 1, 46},
	{"this is a test string \n this is another string", 9, 1, 46},
	{"this * * ***** is a test string", 8, 0, 31},
	{". . . . . . . . . .", 10, 0, 19},
	{`Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod
	tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At
	vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren,
	no sea takimata sanctus est Lorem ipsum dolor sit amet.`, 50, 3, 298},
	{"é", 1, 0, 1},
	{"😀", 1, 0, 1},
	{"👂🏼", 1, 0, 2},
	{"😀😀\n👂🏼", 2, 1, 5},
	{"😀 😀\n👂🏼", 3, 1, 6},
}

func TestCount(t *testing.T) {
	for i, fixture := range tests {
		count := processStream(bufio.NewScanner(strings.NewReader(fixture.input)))

		if count.words != fixture.words {
			t.Error(
				"For test number", i+1,
				"expected", fixture.words,
				"words but",
				"got", count.words,
				"words",
			)
		}

		if count.lines != fixture.lines {
			t.Error(
				"For test number", i+1,
				"expected", fixture.lines,
				"lines but",
				"got", count.lines,
				"lines",
			)
		}

		if count.chars != fixture.chars {
			t.Error(
				"For test number", i+1,
				"expected", fixture.chars,
				"characters but",
				"got", count.chars,
				"characters",
			)
		}
	}
}
