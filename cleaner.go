package main

import (
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
)

var (
	Version               = "3.0.0"
	app                   = kingpin.New("docker-image-cleaner", "Clean up docker images that seem safe to remove.")
	flag_excludes         = app.Flag("exclude", "Leaf images to exclude specified by image:tag").Short('x').PlaceHolder("IMAGE:TAG").Strings()
	flag_deleteLeaf       = app.Flag("delete-dangling", "Delete dangling images").Default("false").Bool()
	flag_deleteDangling   = app.Flag("delete-leaf", "Delete leaf images").Default("false").Bool()
	flag_danglingDuration = app.Flag("dangling-duration", "How far into the past to protect dangling images").Short('d').Default("1h").Duration()
	docker                *client.Client
)

func main() {
	// Stderr is for ERRORS!
	app.Writer(os.Stdout)
	log.SetOutput(os.Stdout)

	app.HelpFlag.Short('h')
	app.Author("Christian Höltje")
	app.Version(Version)
	app.Parse(os.Args[1:])

	initClient()
	cleanLeafImages()
	cleanDanglingImages()
}

func initClient() {
	if os.Getenv("DOCKER_HOST") == "" {
		err := os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
		if err != nil {
			log.Fatalf("Error setting default DOCKER_HOST: %s", err)
		}
	}

	new_client, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Error creating docker client: %s", err)
	}
	docker = new_client
}

func cleanLeafImages() {
	log.Printf("Scanning leaf images...")
	excluded := map[string]struct{}{}

	for _, i := range *flag_excludes {
		excluded[i] = struct{}{}
	}

	leafImages, err := docker.ImageList(types.ImageListOptions{})
	if err != nil {
		log.Fatalf("Error getting docker images: %s", err)
	}

	allImages, err := docker.ImageList(types.ImageListOptions{All: true})
	if err != nil {
		log.Fatalf("Error getting all docker images: %s", err)
	}

	imageTree := make(map[string]types.Image, len(allImages))
	for _, image := range allImages {
		imageTree[image.ID] = image
	}

	containers, err := docker.ContainerList(types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("Error getting docker containers: %s", err)
	}

	imagesToSkip := make(map[string]bool)

	// Find images belonging to containers.
	for _, container := range containers {
		inspected, err := docker.ContainerInspect(container.ID)
		if err != nil {
			log.Printf("Error getting container info for %s: %s", container.ID, err)
			continue
		}

		imagesToSkip[inspected.Image] = true

		for parent := imageTree[inspected.Image].ParentID; len(parent) != 0; parent = imageTree[parent].ParentID {
			imagesToSkip[parent] = true
		}
	}

	// Find images that are "root" images.
	for _, image := range leafImages {
		if imagesToSkip[image.ID] {
			continue
		}

		var topParentId string
		for parentId := image.ParentID; len(parentId) != 0; parentId = imageTree[parentId].ParentID {
			topParentId = parentId
		}

		if len(topParentId) != 0 && !imagesToSkip[topParentId] {
			imagesToSkip[topParentId] = true
			log.Printf("Skipping top parent image %s: %s", topParentId, strings.Join(imageTree[topParentId].RepoTags, ","))
		}
	}

	// Find images in the exclude list.
	for _, image := range leafImages {
		if imagesToSkip[image.ID] {
			continue
		}

		for _, tag := range image.RepoTags {
			if _, ok := excluded[tag]; ok {
				log.Printf("Skipping excluded image %s: %s", image.ID, strings.Join(image.RepoTags, ","))
				imagesToSkip[image.ID] = true
			}
		}
	}

	for _, image := range leafImages {
		if imagesToSkip[image.ID] {
			continue
		}

		nukeImage("leaf", image, *flag_deleteLeaf)
	}
}

func cleanDanglingImages() {
	log.Printf("Scanning dangling images...")
	now := time.Now()

	danglingFilter := filters.NewArgs()
	danglingFilter.Add("dangling", "true")

	danglingImages, err := docker.ImageList(types.ImageListOptions{Filters: danglingFilter})
	if err != nil {
		log.Fatalf("Error getting dangling docker images: %s", err)
	}

	for _, image := range danglingImages {
		created := time.Unix(image.Created, 0)
		if created.Add(*flag_danglingDuration).Before(now) {
			nukeImage("dangling", image, *flag_deleteDangling)
		} else {
			log.Printf("Skipping recent dangling image from %s ago: %s", (now.Sub(created).String()), image.ID)
		}
	}
}

func nukeImage(kind string, image types.Image, really_delete bool) {
	if really_delete {
		log.Printf("Deleting %s image %s: %s", kind, image.ID, strings.Join(image.RepoTags, ","))

		var imagesToNuke []string
		if len(image.RepoTags) <= 1 {
			imagesToNuke = append(imagesToNuke, image.ID)
		} else {
			imagesToNuke = image.RepoTags
		}
		for _, imageIdOrTag := range imagesToNuke {
			_, err := docker.ImageRemove(types.ImageRemoveOptions{ImageID: imageIdOrTag, PruneChildren: true})
			if err != nil {
				log.Printf("Error while removing %s image %s: %s", kind, imageIdOrTag, err)
			}
		}
	} else {
		log.Printf("Would have deleted %s image %s: %s", kind, image.ID, strings.Join(image.RepoTags, ","))
	}
}
