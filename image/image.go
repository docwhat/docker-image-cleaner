package image

import "github.com/docker/engine-api/types"

type Interface interface {
}

type Image struct {
	image types.Image
}

func New(dockerImage types.Image) *Image {
	return &Image{image: dockerImage}
}

func NewList(dockerImages []types.Image) []Interface {
	images := make([]Interface, len(dockerImages))

	for i, dockerImage := range dockerImages {
		images[i] = New(dockerImage)
	}
	return images
}
