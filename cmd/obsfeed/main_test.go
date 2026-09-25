package main

import (
	"bytes"
	"testing"
)

func TestForwardPCMEmitsOrderedHundredMillisecondChunks(t *testing.T) {
	input := bytes.Repeat([]byte{0x01, 0x02}, audioChunkBytes/2+1)
	var sequences []int64
	var sizes []int
	lastFirstByte := byte(0)
	count, err := forwardPCM(bytes.NewReader(input), func(sequence int64, chunk []byte) error {
		sequences = append(sequences, sequence)
		sizes = append(sizes, len(chunk))
		lastFirstByte = chunk[0]
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || len(sequences) != 2 || sequences[0] != 1 || sequences[1] != 2 {
		t.Fatalf("sequences = %v, count = %d", sequences, count)
	}
	if sizes[0] != audioChunkBytes || sizes[1] != 2 || lastFirstByte != 0x01 {
		t.Fatalf("chunk sizes = %v; expected [%d 2]", sizes, audioChunkBytes)
	}
}

func TestForwardPCMRejectsOddByteTail(t *testing.T) {
	count, err := forwardPCM(bytes.NewReader([]byte{0x01}), func(int64, []byte) error { return nil })
	if err == nil || count != 0 {
		t.Fatalf("forwardPCM = (%d, %v), want an odd-byte error", count, err)
	}
}
