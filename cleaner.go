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
	Version             = "4.0.3"
	app                 = kingpin.New("docker-image-cleaner", "Clean up docker images that seem safe to remove.")
	flag_excludes       = app.Flag("exclude", "Leaf images to exclude specified by image:tag").Short('x').PlaceHolder("IMAGE:TAG").Strings()
	flag_deleteDangling = app.Flag("delete-dangling", "Delete dangling images").Default("false").Bool()
	flag_deleteLeaf     = app.Flag("delete-leaf", "Delete leaf images").Default("false").Bool()
	flag_safetyDuration = app.Flag("safety-duration", "Don't delete any images created in the last DUR time").Short('d').PlaceHolder("DUR").Default("1h").HintOptions("30m", "1h", "24h").Duration()
	now                 = time.Unix(time.Now().Unix(), 0) // Now without sub-seconds.
	imagesToSkip        = make(map[string]bool)

	docker     *client.Client
	imagesById map[string]types.Image
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

	allImages, err := docker.ImageList(types.ImageListOptions{All: true})
	if err != nil {
		log.Fatalf("Error getting all docker images: %s", err)
	}

	imagesById = make(map[string]types.Image, len(allImages))
	for _, image := range allImages {
		imagesById[image.ID] = image
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
	leafImages, err := docker.ImageList(types.ImageListOptions{})
	if err != nil {
		log.Fatalf("Error getting docker images: %s", err)
	}

	pruneContainerImages()

	// Find images that are "root" images.
	for _, image := range leafImages {
		if imagesToSkip[image.ID] {
			continue
		}

		for parentId := image.ParentID; len(parentId) != 0; parentId = imagesById[parentId].ParentID {
			image := imagesById[parentId]
			if len(image.RepoTags) == 1 && image.RepoTags[0] == "<none>:<none>" {
				continue
			}
			if len(image.RepoDigests) == 1 && image.RepoDigests[0] == "<none>@<none>" {
				continue
			}

			if !imagesToSkip[parentId] {
				imagesToSkip[parentId] = true
				log.Printf("Skipping tagged parent image %s: %s", shortImageDigest(parentId), strings.Join(image.RepoTags, ","))
			}
		}
	}

	pruneExcludedImages(leafImages)
	pruneUnsafeImages(leafImages)

	for _, image := range leafImages {
		if imagesToSkip[image.ID] {
			continue
		}

		nukeImage("leaf", image, *flag_deleteLeaf)
	}
}

func cleanDanglingImages() {
	danglingFilter := filters.NewArgs()
	danglingFilter.Add("dangling", "true")

	danglingImages, err := docker.ImageList(types.ImageListOptions{Filters: danglingFilter})
	if err != nil {
		log.Fatalf("Error getting dangling docker images: %s", err)
	}

	pruneContainerImages()
	pruneUnsafeImages(danglingImages)

	for _, image := range danglingImages {
		if imagesToSkip[image.ID] {
			continue
		}

		nukeImage("dangling", image, *flag_deleteDangling)
	}
}

func pruneContainerImages() {
	containers, err := docker.ContainerList(types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("Error getting docker containers: %s", err)
	}

	// Find images belonging to containers.
	for _, container := range containers {
		inspected, err := docker.ContainerInspect(container.ID)
		if err != nil {
			log.Printf("Error getting container info for %s: %s", container.ID, err)
			continue
		}

		for parent := inspected.Image; len(parent) != 0; parent = imagesById[parent].ParentID {
			if !imagesToSkip[parent] {
				imagesToSkip[parent] = true
				log.Printf("Skipping in use container image %s: %s", shortImageDigest(parent), strings.Join(imagesById[parent].RepoTags, ","))
			}
		}
	}
}

func pruneExcludedImages(images []types.Image) {
	excluded := asSet(*flag_excludes)

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
		if age < *flag_safetyDuration {
			if !imagesToSkip[image.ID] {
				log.Printf("Skipping recent image %s: only %s old", shortImageDigest(image.ID), age)
				imagesToSkip[image.ID] = true
			}
		}
	}
}

func nukeImage(kind string, image types.Image, really_delete bool) {
	if really_delete {
		log.Printf("Deleting %s image %s: %s", kind, shortImageDigest(image.ID), strings.Join(image.RepoTags, ","))

		var imagesToNuke []string
		if len(image.RepoTags) <= 1 {
			imagesToNuke = append(imagesToNuke, image.ID)
		} else {
			imagesToNuke = image.RepoTags
		}
		for _, imageIdOrTag := range imagesToNuke {
			_, err := docker.ImageRemove(types.ImageRemoveOptions{ImageID: imageIdOrTag, PruneChildren: true})
			if err != nil {
				log.Printf("Error while removing %s image %s: %s", kind, shortImageDigest(imageIdOrTag), err)
			}
		}
	} else {
		log.Printf("Would have deleted %s image %s: %s", kind, shortImageDigest(image.ID), strings.Join(image.RepoTags, ","))
	}
}
