package client

import (
	"log"
	"os"

	"docwhat.org/docker-image-cleaner/image"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
)

// Interface for interacting with a docker client.
type Interface interface {
	AllImages() []image.Interface
	DanglingImages() []image.Interface
	TaggedOrphanImages() []image.Interface
}

// A Client wrapping docker client.
type Client struct {
	docker client.APIClient
	ctx    context.Context
}

// Ensures we have DOCKER_HOST set.
func init() {
	if os.Getenv("DOCKER_HOST") == "" {
		err := os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
		if err != nil {
			log.Fatalf("Error setting default DOCKER_HOST: %s", err)
		}
	}
}

// New creates a Client struct.
func New() *Client {
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Error creating docker client: %s", err)
	}
	return &Client{docker: dockerClient, ctx: context.Background()}
}

// AllImages returns all images in docker.
//
// This is the same as running `docker ps --all`
func (c Client) AllImages() []image.Interface {
	options := types.ImageListOptions{All: true}
	dockerImages, err := c.docker.ImageList(c.ctx, options)
	if err != nil {
		log.Fatalf("Error getting all images: %s", err)
	}
	return image.NewList(dockerImages)
}

// DanglingImages returns all images with no parents that have no tags.
//
// This is the same as running `docker ps --filter dangling=true`
func (c Client) DanglingImages() []image.Interface {
	options := types.ImageListOptions{Filters: c.danglingFilter()}

	dockerImages, err := c.docker.ImageList(c.ctx, options)
	if err != nil {
		log.Fatalf("Error getting dangling images: %s", err)
	}
	return image.NewList(dockerImages)
}

// TaggedOrphanImages returns all images that have no parents but have tags.
//
// This is the same as running `docker ps`
func (c Client) TaggedOrphanImages() []image.Interface {
	options := types.ImageListOptions{}

	dockerImages, err := c.docker.ImageList(c.ctx, options)
	if err != nil {
		log.Fatalf("Error getting orphaned images: %s", err)
	}
	return image.NewList(dockerImages)
}

func (c Client) danglingFilter() filters.Args {
	filter := filters.NewArgs()
	filter.Add("dangling", "true")
	return filter
}

// Ensure that Client always implements Interface
var _ Interface = &Client{}
