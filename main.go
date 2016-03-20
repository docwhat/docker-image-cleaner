package main

import (
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/alecthomas/kingpin"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"
)

var (
	version            = "4.0.4"
	app                = kingpin.New("docker-image-cleaner", "Clean up docker images that seem safe to remove.")
	flagExcludes       = app.Flag("exclude", "Leaf images to exclude specified by image:tag").Short('x').PlaceHolder("IMAGE:TAG").Strings()
	flagDeleteDangling = app.Flag("delete-dangling", "Delete dangling images").Default("false").Bool()
	flagDeleteLeaf     = app.Flag("delete-leaf", "Delete leaf images").Default("false").Bool()
	flagSafetyDuration = app.Flag("safety-duration", "Don't delete any images created in the last DUR time").Short('d').PlaceHolder("DUR").Default("1h").HintOptions("30m", "1h", "24h").Duration()
	now                = time.Unix(time.Now().Unix(), 0) // Now without sub-seconds.
	imagesToSkip       = make(map[string]bool)

	docker     *client.Client
	imagesByID map[string]types.Image
)

func main() {
	// Stderr is for ERRORS!
	app.Writer(os.Stdout)
	log.SetOutput(os.Stdout)

	app.HelpFlag.Short('h')
	app.Author("Christian Höltje")
	app.Version(version)
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

	newClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("Error creating docker client: %s", err)
	}
	docker = newClient

	allImages, err := docker.ImageList(context.Background(), types.ImageListOptions{All: true})
	if err != nil {
		log.Fatalf("Error getting all docker images: %s", err)
	}

	imagesByID = make(map[string]types.Image, len(allImages))
	for _, image := range allImages {
		imagesByID[image.ID] = image
	}
}

func asSet(elements []string) map[string]struct{} {
	s := make(map[string]struct{}, len(elements))
	present := struct{}{}
	for _, e := range elements {
		s[e] = present
	}
	return s
}

func ageOf(image types.Image) time.Duration {
	return now.Sub(time.Unix(image.Created, 0))
}

func shortImageDigest(id string) string {
	if strings.HasPrefix(id, "sha256:") {
		return id[7:14]
	}
	return id
}

func cleanLeafImages() {
	leafImages, err := docker.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Fatalf("Error getting docker images: %s", err)
	}

	pruneContainerImages()

	// Find images that are "root" images.
	for _, image := range leafImages {
		if imagesToSkip[image.ID] {
			continue
		}

		for parentID := image.ParentID; len(parentID) != 0; parentID = imagesByID[parentID].ParentID {
			image := imagesByID[parentID]
			if len(image.RepoTags) == 1 && image.RepoTags[0] == "<none>:<none>" {
				continue
			}
			if len(image.RepoDigests) == 1 && image.RepoDigests[0] == "<none>@<none>" {
				continue
			}

			if !imagesToSkip[parentID] {
				imagesToSkip[parentID] = true
				log.Printf("Skipping tagged parent image %s: %s", shortImageDigest(parentID), strings.Join(image.RepoTags, ","))
			}
		}
	}

	pruneExcludedImages(leafImages)
	pruneUnsafeImages(leafImages)

	for _, image := range leafImages {
		if imagesToSkip[image.ID] {
			continue
		}

		nukeImage("leaf", image, *flagDeleteLeaf)
	}
}

func cleanDanglingImages() {
	danglingFilter := filters.NewArgs()
	danglingFilter.Add("dangling", "true")

	danglingImages, err := docker.ImageList(context.Background(), types.ImageListOptions{Filters: danglingFilter})
	if err != nil {
		log.Fatalf("Error getting dangling docker images: %s", err)
	}

	pruneContainerImages()
	pruneUnsafeImages(danglingImages)

	for _, image := range danglingImages {
		if imagesToSkip[image.ID] {
			continue
		}

		nukeImage("dangling", image, *flagDeleteDangling)
	}
}

func pruneContainerImages() {
	containers, err := docker.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("Error getting docker containers: %s", err)
	}

	// Find images belonging to containers.
	for _, container := range containers {
		inspected, err := docker.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			log.Printf("Error getting container info for %s: %s", container.ID, err)
			continue
		}

		for parent := inspected.Image; len(parent) != 0; parent = imagesByID[parent].ParentID {
			if !imagesToSkip[parent] {
				imagesToSkip[parent] = true
				log.Printf("Skipping in use container image %s: %s", shortImageDigest(parent), strings.Join(imagesByID[parent].RepoTags, ","))
			}
		}
	}
}

func pruneExcludedImages(images []types.Image) {
	excluded := asSet(*flagExcludes)

	for _, image := range images {
		if imagesToSkip[image.ID] {
			continue
		}

		for _, tag := range image.RepoTags {
			if _, ok := excluded[tag]; ok {
				log.Printf("Skipping excluded image %s: %s", shortImageDigest(image.ID), strings.Join(image.RepoTags, ","))
				imagesToSkip[image.ID] = true
			}
		}
	}
}

func pruneUnsafeImages(images []types.Image) {
	for _, image := range images {
		age := ageOf(image)
		if age < *flagSafetyDuration {
			if !imagesToSkip[image.ID] {
				log.Printf("Skipping recent image %s: only %s old", shortImageDigest(image.ID), age)
				imagesToSkip[image.ID] = true
			}
		}
	}
}

func nukeImage(kind string, image types.Image, reallyDelete bool) {
	if reallyDelete {
		log.Printf("Deleting %s image %s: %s", kind, shortImageDigest(image.ID), strings.Join(image.RepoTags, ","))

		var imagesToNuke []string
		if len(image.RepoTags) <= 1 {
			imagesToNuke = append(imagesToNuke, image.ID)
		} else {
			imagesToNuke = image.RepoTags
		}
		for _, imageIDOrTag := range imagesToNuke {
			_, err := docker.ImageRemove(context.Background(), types.ImageRemoveOptions{ImageID: imageIDOrTag, PruneChildren: true})
			if err != nil {
				log.Printf("Error while removing %s image %s: %s", kind, shortImageDigest(imageIDOrTag), err)
			}
		}
	} else {
		log.Printf("Would have deleted %s image %s: %s", kind, shortImageDigest(image.ID), strings.Join(image.RepoTags, ","))
	}
}
