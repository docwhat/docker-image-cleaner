package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
)

var flag_exclude string
var flag_dryRun bool
var flag_danglingDuration time.Duration
var docker *client.Client

func main() {
	flag.StringVar(&flag_exclude, "exclude", "", "images to exclude, image:tag[,image:tag]")
	flag.BoolVar(&flag_dryRun, "dry-run", false, "just list containers to remove")
	flag.DurationVar(&flag_danglingDuration, "dangling-duration", time.Hour, "how far into the past to protect dangling images")
	flag.Parse()

	initClient()
	cleanLeafImages()
	cleanDanglingImages()
}

func initClient() {
	if os.Getenv("DOCKER_HOST") == "" {
		err := os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
		if err != nil {
			log.Fatalf("error setting default DOCKER_HOST: %s", err)
		}
	}

	new_client, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("error creating docker client: %s", err)
	}
	docker = new_client
}

func cleanLeafImages() {
	excluded := map[string]struct{}{}

	if len(flag_exclude) > 0 {
		for _, i := range strings.Split(flag_exclude, ",") {
			excluded[i] = struct{}{}
		}
	}

	leafImages, err := docker.ImageList(types.ImageListOptions{})
	if err != nil {
		log.Fatalf("error getting docker images: %s", err)
	}

	allImages, err := docker.ImageList(types.ImageListOptions{All: true})
	if err != nil {
		log.Fatalf("error getting all docker images: %s", err)
	}

	imageTree := make(map[string]types.Image, len(allImages))
	for _, image := range allImages {
		imageTree[image.ID] = image
	}

	containers, err := docker.ContainerList(types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("error getting docker containers: %s", err)
	}

	imagesToSkip := make(map[string]bool)

	for _, container := range containers {
		inspected, err := docker.ContainerInspect(container.ID)
		if err != nil {
			log.Printf("error getting container info for %s: %s", container.ID, err)
			continue
		}

		imagesToSkip[inspected.Image] = true

		for parent := imageTree[inspected.Image].ParentID; len(parent) != 0; parent = imageTree[parent].ParentID {
			imagesToSkip[parent] = true
		}
	}

	for _, image := range leafImages {
		for _, tag := range image.RepoTags {
			if _, ok := excluded[tag]; ok {
				log.Printf("Skipping excluded image %s: %s", image.ID, strings.Join(image.RepoTags, ","))
				imagesToSkip[image.ID] = true
			}
		}
	}

	for _, image := range leafImages {
		if _, ok := imagesToSkip[image.ID]; !ok {
			nukeImage(image)
		}
	}
}

func cleanDanglingImages() {
	log.Printf("Scanning dangling images...")
	danglingFilter := filters.NewArgs()
	danglingFilter.Add("dangling", "true")

	danglingImages, err := docker.ImageList(types.ImageListOptions{Filters: danglingFilter})
	if err != nil {
		log.Fatalf("error getting dangling docker images: %s", err)
	}

	for _, image := range danglingImages {
		created := time.Unix(image.Created, 0)
		if created.Add(flag_danglingDuration).Before(time.Now()) {
			nukeImage(image)
		}
	}
}

func nukeImage(image types.Image) {
	if flag_dryRun {
		log.Printf("Would have deleted image %s: %s", image.ID, strings.Join(image.RepoTags, ","))
	} else {
		log.Printf("Deleting leaf image %s: %s", image.ID, strings.Join(image.RepoTags, ","))

		var imagesToNuke []string
		if len(image.RepoTags) <= 1 {
			imagesToNuke = append(imagesToNuke, image.ID)
		} else {
			imagesToNuke = image.RepoTags
		}
		for _, imageIdOrTag := range imagesToNuke {
			_, err := docker.ImageRemove(types.ImageRemoveOptions{ImageID: imageIdOrTag, PruneChildren: true})
			if err != nil {
				log.Printf("error while removing image %s: %s", imageIdOrTag, err)
			}
		}
	}
}
