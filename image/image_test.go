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

// **********************************************
// ShortID
func TestNullShortID(t *testing.T) {
	assert := assert.New(t)
	i := Image{}

	assert.Equal("", i.ShortID())

	i.ID = "sha256:ffe2bc3a35a361004484b73c5797bd6254dc18cf93a5a48e5be7df784a71711e"
	assert.Equal("ffe2bc3", i.ShortID())

	i.ID = "fred"
	assert.Equal("fred", i.ShortID())
}

func TestShortParentID(t *testing.T) {
	assert := assert.New(t)
	i := Image{}

	assert.Equal("", i.ShortParentID())

	i.ParentID = "sha256:d8be53875f1e4d291c68a0b31a00a4eb2a171d37efdad3b551d1aa41da7841a6"
	assert.Equal("d8be538", i.ShortParentID())

	i.ParentID = "fred"
	assert.Equal("fred", i.ShortParentID())
}

func TestIsOrphan(t *testing.T) {
	assert := assert.New(t)
	i := Image{}

	i.ParentID = ""
	assert.True(i.IsOrphan())

	i.ParentID = "sha256:5b4e987b9b0945abcef0ddb745fd70f8f3110a34a3424f215c62c134deecb80f"
	assert.False(i.IsOrphan())
}

func TestHasTags(t *testing.T) {
	assert := assert.New(t)
	i := Image{}

	assert.False(i.HasTags())

	i.RepoTags = append(i.RepoTags, "fred:1.2.3")
	assert.True(i.HasTags())

	i.RepoTags[0] = "<none>:<none>"
	assert.False(i.HasTags(), "<none>:<none> should not be considered a tag")
}

func TestHasDigests(t *testing.T) {
	assert := assert.New(t)
	i := Image{}

	assert.False(i.HasDigests())

	i.RepoDigests = append(i.RepoDigests, "fred@foobar")
	assert.True(i.HasDigests(), "There is a digest so it should be true")

	i.RepoDigests[0] = "<none>@<none>"
	assert.False(i.HasDigests(), "<none>@<none> should not be considered a digest")
}
