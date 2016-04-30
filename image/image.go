package image

import (
	"strings"
	"time"

	"github.com/docker/engine-api/types"
)

var (
	now = time.Unix(time.Now().Unix(), 0) // Now without sub-seconds.
)

// The Interface for manipulating container images.
type Interface interface {
	AgeOf() time.Duration
	ShortID() string
	ShortParentID() string
}

// Image provides methods for manipulating docker images.
type Image struct {
	Interface
	image types.Image
}

// New creates a new Image from docker images.
func New(dockerImage types.Image) *Image {
	return &Image{image: dockerImage}
}

// NewList creates a list of Images from docker images.
func NewList(dockerImages []types.Image) []Interface {
	images := make([]Interface, len(dockerImages))

	for i, dockerImage := range dockerImages {
		images[i] = New(dockerImage)
	}
	return images
}

// AgeOf returns how long since the image was created.
func (i Image) AgeOf() time.Duration {
	return now.Sub(time.Unix(i.image.Created, 0))
}

func (i Image) ShortID() string {
	return shortenDigest(i.image.ID)
}

func (i Image) ShortParentID() string {
	return shortenDigest(i.image.ParentID)
}

func shortenDigest(digest string) string {
	if strings.HasPrefix(digest, "sha256:") {
		return digest[7:14]
	}
	return digest
}
