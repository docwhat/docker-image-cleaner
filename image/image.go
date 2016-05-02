package image

import (
	"fmt"
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
	IsOrphan() bool
	HasTags() bool
	HasDigests() bool
}

// Image provides methods for manipulating docker images.
type Image struct {
	types.Image
}

// New creates a new Image from docker images.
func New(dockerImage types.Image) *Image {
	return &Image{dockerImage}
}

// NewList creates a list of Images from docker images.
func NewList(dockerImages []types.Image) []Image {
	images := make([]Image, len(dockerImages))

	for i, dockerImage := range dockerImages {
		images[i] = *New(dockerImage)
	}
	return images
}

// IsOrphan returns true if the image has no parents.
func (i *Image) IsOrphan() bool {
	return "" == i.ParentID
}

// HasTags returns true if any tags (other than `<none>:<none>`) are present.
func (i *Image) HasTags() bool {
	switch len(i.RepoTags) {
	case 0:
		return false
	case 1:
		return i.RepoTags[0] != "<none>:<none>"
	}
	return true
}

// HasDigests returns true if any digests (other than `<none>@<none>`) are present.
func (i *Image) HasDigests() bool {
	switch len(i.RepoDigests) {
	case 0:
		return false
	case 1:
		return i.RepoDigests[0] != "<none>@<none>"
	}
	return true
}

// AgeOf returns how long since the image was created.
func (i Image) AgeOf() time.Duration {
	return now.Sub(time.Unix(i.Created, 0))
}

// Pretty print info about the image.
func (i Image) String() string {
	if len(i.RepoTags) > 0 {
		return fmt.Sprintf("%s: %s", i.ShortID(), strings.Join(i.RepoTags, ", "))
	}
	return i.ShortID()
}

// ShortID provides a shortened form of the ID digest.
func (i Image) ShortID() string {
	return shortenDigest(i.ID)
}

// ShortParentID provides a shortened form of the ParentID digest.
func (i Image) ShortParentID() string {
	return shortenDigest(i.ParentID)
}

func shortenDigest(digest string) string {
	if strings.HasPrefix(digest, "sha256:") {
		return digest[7:14]
	}
	return digest
}

// Ensure that Image always implements Interface
var _ Interface = &Image{}
