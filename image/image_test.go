package image

import (
	"testing"

	"github.com/docker/engine-api/types"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	var image types.Image
	New(image)
}

func TestShortenDigest(t *testing.T) {
	sha := "sha256:d8be53875f1e4d291c68a0b31a00a4eb2a171d37efdad3b551d1aa41da7841a6"
	got := shortenDigest(sha)
	want := "d8be538"
	assert.Equal(t, want, got)
}
