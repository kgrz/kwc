package main

import "testing"

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
	{"this is a test string", 5, 0, 22},
	{"this   is          a     test string", 5, 0, 37},
	{"this	is			a		test string", 5, 0, 25},
	{"this is a test string\nthis is another strirg", 9, 1, 45},
	{"this is a test string\n\n\n\n\n\nthis is another string", 9, 6, 50},
	{"this is a test string\n this is another string", 9, 1, 46},
	{"this is a test str-\ning this is another string", 10, 1, 47},
	{"this is a test string \n this is another string", 9, 1, 47},
	{"this * * ***** is a test string", 8, 0, 32},
	{". . . . . . . . . .", 10, 0, 20},
	{`Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod
	tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At
	vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren,
	no sea takimata sanctus est Lorem ipsum dolor sit amet.`, 50, 3, 299},
	{"é", 1, 0, 2},
	{"😀", 1, 0, 2},
	{"👂🏼", 1, 0, 3},
	{"😀😀\n👂🏼", 2, 1, 6},
	{"😀 😀\n👂🏼", 3, 1, 7},
}

func TestCount(t *testing.T) {
	for i, fixture := range tests {
		count := countBuffer([]byte(fixture.input))

		if count.Words != fixture.words {
			t.Error(
				"For test number", i+1,
				"expected", fixture.words,
				"words but",
				"got", count.Words,
				"words",
			)
		}

		if count.Lines != fixture.lines {
			t.Error(
				"For test number", i+1,
				"expected", fixture.lines,
				"lines but",
				"got", count.Lines,
				"lines",
			)
		}

		if count.Chars != fixture.chars {
			t.Error(
				"For test number", i+1,
				"expected", fixture.chars,
				"characters but",
				"got", count.Chars,
				"characters",
			)
		}
	}
}
