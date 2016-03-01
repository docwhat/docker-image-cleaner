package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
)

func main() {
	exclude := flag.String("exclude", "", "images to exclude, image:tag[,image:tag]")
	dryRun := flag.Bool("dry-run", false, "just list containers to remove")
	flag.Parse()

	if os.Getenv("DOCKER_HOST") == "" {
		err := os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
		if err != nil {
			log.Fatalf("error setting default DOCKER_HOST: %s", err)
		}
	}

	excluded := map[string]struct{}{}

	if len(*exclude) > 0 {
		for _, i := range strings.Split(*exclude, ",") {
			excluded[i] = struct{}{}
		}
	}

	docker, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("error creating docker client: %s", err)
	}

	topImages, err := docker.ImageList(types.ImageListOptions{})
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

	used := make(map[string]struct{}, len(containers))

	for _, container := range containers {
		inspected, err := docker.ContainerInspect(container.ID)
		if err != nil {
			log.Printf("error getting container info for %s: %s", container.ID, err)
			continue
		}

		used[inspected.Image] = struct{}{}

		for parent := imageTree[inspected.Image].ParentID; len(parent) != 0; parent = imageTree[parent].ParentID {
			used[parent] = struct{}{}
		}
	}

	removalImageLoop:
	for _, image := range topImages {
		if _, ok := used[image.ID]; !ok {
			for _, tag := range image.RepoTags {
				if _, ok := excluded[tag]; ok {
					log.Printf("Skipping %s: %s", image.ID, strings.Join(image.RepoTags, ","))
					continue removalImageLoop
				}
			}

			log.Printf("Going to remove image %s: %s", image.ID, strings.Join(image.RepoTags, ","))

			if !*dryRun {
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
	}
}
