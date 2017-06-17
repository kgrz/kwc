package main

import (
	"reflect"
	"testing"
)

func Test_bufferStates_reduce(t *testing.T) {
	tests := []struct {
		name string
		bxs  bufferStates
		want bufferState
	}{
		{
			"first buffer ends in space, second buffer starts with space",
			bufferStates{
				countBuffer([]byte("this is ")),
				countBuffer([]byte(" something else")),
			},
			bufferState{words: 4, chars: 23, lines: 0, firstChar: 't', lastChar: 'e'},
		},
		{
			"first buffer ends in space, second buffer does not start with space",
			bufferStates{
				countBuffer([]byte("this is ")),
				countBuffer([]byte("something else")),
			},
			bufferState{words: 4, chars: 22, lines: 0, firstChar: 't', lastChar: 'e'},
		},
		{
			"first and second buffers don't end or start in space",
			bufferStates{
				countBuffer([]byte("this some")),
				countBuffer([]byte("thing else")),
			},
			bufferState{words: 3, chars: 19, lines: 0, firstChar: 't', lastChar: 'e'},
		},
		{
			"first buffer has a new line",
			bufferStates{
				countBuffer([]byte("this \n some")),
				countBuffer([]byte("thing else")),
			},
			bufferState{words: 3, chars: 21, lines: 1, firstChar: 't', lastChar: 'e'},
		},
		{
			"first buffer has a new line at the end, second buffer has one at the beginning",
			bufferStates{
				countBuffer([]byte("this some\n\n")),
				countBuffer([]byte("\n\nthing else")),
			},
			bufferState{words: 4, chars: 23, lines: 4, firstChar: 't', lastChar: 'e'},
		},
		{
			"double buffers, only emojis",
			bufferStates{
				countBuffer([]byte("😀 😀👂🏼")),
				countBuffer([]byte("😀 😀👂🏼")),
			},
			bufferState{words: 3, chars: 10, lines: 0, firstChar: 240, lastChar: 188},
		},
		{
			"single buffer, only emojis. should be same as the one above",
			bufferStates{
				countBuffer([]byte("😀 😀👂🏼😀 😀👂🏼")),
			},
			bufferState{words: 3, chars: 10, lines: 0, firstChar: 240, lastChar: 188},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bxs.reduce(bufferState{}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bufferStates.reduce() = %v, want %v", got, tt.want)
			}
		})
	}
}
