package thema

import (
	"testing"
	"text/template/parse"
)

func TestCountIdentifierHandlesNilPipe(t *testing.T) {
	var pipe *parse.PipeNode
	if got := countIdentifier(pipe, "slot"); got != 0 {
		t.Fatalf("countIdentifier() = %d, want 0", got)
	}
}
