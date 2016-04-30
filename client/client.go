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

func init() {
	if os.Getenv("DOCKER_HOST") == "" {
		err := os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
		if err != nil {
			log.Fatalf("Error setting default DOCKER_HOST: %s", err)
		}
	}
}

type Interface interface {
	AllImages() ([]image.Interface, error)
	DanglingImages() ([]image.Interface, error)
	ParentlessImages() ([]image.Interface, error)
}

type Client struct {
	docker *client.Client
	ctx    context.Context
}

func New() *Client {
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Error creating docker client: %s", err)
	}
	return &Client{docker: dockerClient, ctx: context.Background()}
}

func (c Client) AllImages() []image.Interface {
	options := types.ImageListOptions{All: true}
	dockerImages, err := c.docker.ImageList(c.ctx, options)
	if err != nil {
		log.Fatalf("Error getting all images: %s", err)
	}
	return image.NewList(dockerImages)
}

func (c Client) DanglingImages() []image.Interface {
	options := types.ImageListOptions{Filters: c.danglingFilter()}

	dockerImages, err := c.docker.ImageList(c.ctx, options)
	if err != nil {
		log.Fatalf("Error getting dangling images: %s", err)
	}
	return image.NewList(dockerImages)
}

func (c Client) OrphanedImages() []image.Interface {
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
