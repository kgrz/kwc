package main

import (
	"reflect"
	"testing"
)

func Test_bufferStates_reduce(t *testing.T) {
	tests := []struct {
		name string
		bxs  []chunk
		want chunk
	}{
		{
			"first buffer ends in space, second buffer starts with space",
			[]chunk{
				countBuffer([]byte("this is ")),
				countBuffer([]byte(" something else")),
			},
			chunk{words: 4, chars: 23, lines: 0, firstByte: 't', lastByte: 'e'},
		},
		{
			"first buffer ends in space, second buffer does not start with space",
			[]chunk{
				countBuffer([]byte("this is ")),
				countBuffer([]byte("something else")),
			},
			chunk{words: 4, chars: 22, lines: 0, firstByte: 't', lastByte: 'e'},
		},
		{
			"first and second buffers don't end or start in space",
			[]chunk{
				countBuffer([]byte("this some")),
				countBuffer([]byte("thing else")),
			},
			chunk{words: 3, chars: 19, lines: 0, firstByte: 't', lastByte: 'e'},
		},
		{
			"first buffer has a new line",
			[]chunk{
				countBuffer([]byte("this \n some")),
				countBuffer([]byte("thing else")),
			},
			chunk{words: 3, chars: 21, lines: 1, firstByte: 't', lastByte: 'e'},
		},
		{
			"first buffer has a new line at the end, second buffer has one at the beginning",
			[]chunk{
				countBuffer([]byte("this some\n\n")),
				countBuffer([]byte("\n\nthing else")),
			},
			chunk{words: 4, chars: 23, lines: 4, firstByte: 't', lastByte: 'e'},
		},
		{
			"double buffers, only emojis",
			[]chunk{
				countBuffer([]byte("😀 😀👂🏼")),
				countBuffer([]byte("😀 😀👂🏼")),
			},
			chunk{words: 3, chars: 10, lines: 0, firstByte: 240, lastByte: 188},
		},
		{
			"single buffer, only emojis. should be same as the one above",
			[]chunk{
				countBuffer([]byte("😀 😀👂🏼😀 😀👂🏼")),
			},
			chunk{words: 3, chars: 10, lines: 0, firstByte: 240, lastByte: 188},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reduce(tt.bxs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reduce([]chunk) = %v, want %v", got, tt.want)
			}
		})
	}
}
