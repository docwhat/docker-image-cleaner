package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	client "docwhat.org/docker-image-cleaner/client"
	image "docwhat.org/docker-image-cleaner/image"

	"github.com/alecthomas/kingpin"
)

const (
	// Keep -- the image will be kept.
	Keep = iota
	// DeleteDangling -- the image will be deleted as a dangling image.
	DeleteDangling
	// DeleteLeaf -- the image will be deleted as a leaf.
	DeleteLeaf
)

var (
	version            = "4.0.5"
	flagExcludes       = kingpin.Flag("exclude", "Leaf images to exclude specified by image:tag").Short('x').PlaceHolder("IMAGE:TAG").Strings()
	flagDeleteDangling = kingpin.Flag("delete-dangling", "Delete dangling images").Default("false").Bool()
	flagDeleteLeaf     = kingpin.Flag("delete-leaf", "Delete leaf images").Default("false").Bool()
	flagSafetyDuration = kingpin.Flag("safety-duration", "Don't delete any images created in the last DUR time").Short('d').PlaceHolder("DUR").Default("1h").HintOptions("30m", "1h", "24h").Duration()
)

type meta struct {
	image          image.Image
	isDangling     bool
	isExcluded     bool
	isInUse        bool
	isTaggedOrphan bool
	isSafe         bool
	doDelete       int
}

func (m *meta) String() string {
	return fmt.Sprintf(
		"d: %-5t  e: %-5t  r: %-5t  to: %-5t  s:%-5t  X:%d",
		m.isDangling,
		m.isExcluded,
		m.isInUse,
		m.isTaggedOrphan,
		m.isSafe,
		m.doDelete,
	)
}

func main() {
	// Stderr is for ERRORS!
	kingpin.CommandLine.Writer(os.Stdout)

	kingpin.HelpFlag.Short('h')
	kingpin.CommandLine.Help = "Clean up docker images that seem safe to remove."
	kingpin.CommandLine.Author("Christian Höltje")
	kingpin.Version(version)
	kingpin.Parse()

	docker := client.New()

	excluded := asSet(*flagExcludes)

	// populate all images
	allImages := docker.AllImages()
	imagesByID := make(map[string]*meta, len(allImages))
	for _, image := range allImages {
		meta := &meta{image: image}
		for _, tag := range image.RepoTags {
			if _, ok := excluded[tag]; ok {
				meta.isExcluded = true
				break
			}
		}
		imagesByID[image.ID] = meta
	}

	// mark dangling information
	for _, image := range docker.DanglingImages() {
		m := imagesByID[image.ID]
		if m != nil {
			m.isDangling = true
		}
	}

	// mark tagged orphan information
	for _, image := range docker.TaggedOrphanImages() {
		m := imagesByID[image.ID]
		if m != nil {
			m.isTaggedOrphan = true
		}
	}

	// mark images in a container
	for _, imageID := range docker.AllContainerImageIDs() {
		m := imagesByID[imageID]
		if m != nil {
			m.isInUse = true
		}
	}

	// mark images that are too recent
	for _, m := range imagesByID {
		if m.image.AgeOf() < *flagSafetyDuration {
			m.isSafe = true
		}
	}

	// mark the parents of keep as Keep
	// FIXME: This isn't correct... is it even needed?
	for _, m := range imagesByID {
		for parentID := m.image.ParentID; len(parentID) != 0; parentID = imagesByID[parentID].image.ParentID {
			parentMeta := imagesByID[parentID]
			if parentMeta.doDelete == Keep {
				break
			}
			parentMeta.doDelete = Keep
		}
	}

	for _, m := range imagesByID {
		if m.isExcluded || m.isInUse || m.isSafe {
			continue
		}
		if m.isDangling {
			m.doDelete = DeleteDangling
		} else if m.isTaggedOrphan {
			m.doDelete = DeleteLeaf
		}
	}

	ids := make([]string, len(imagesByID))
	i := 0
	for id := range imagesByID {
		ids[i] = id
		i++
	}
	sort.Strings(ids)

	for _, id := range ids {
		m := imagesByID[id]
		if m == nil {
			log.Fatalf("Internal error! %v", id)
		}
		switch m.doDelete {
		case Keep:
			fmt.Printf("Skipping tagged parent image %s\n", m.image)
		case DeleteDangling:
			// fmt.Printf("Deleting dangling %s\n", m.image)
			nukeImage("dangling", m.image, *flagDeleteDangling)
		case DeleteLeaf:
			// fmt.Printf("Deleting leaf     %s\n", m.image)
			nukeImage("leaf", m.image, *flagDeleteLeaf)
		}
		// fmt.Printf("NARF %s %s\n", m, m.image)
	}
	// for _, image := range docker.DanglingImages() {
	//   fmt.Printf("D %s %s\n", image.ShortID(), image.IsDangling())
	// }

	// for _, image := range docker.AllImages() {
	//   fmt.Printf("A %s %s\n", image.ShortID(), image.IsDangling())
	// }
	// // cleanLeafImages()
	// cleanDanglingImages()
}

// func initClient() {
//   if os.Getenv("DOCKER_HOST") == "" {
//     err := os.Setenv("DOCKER_HOST", "unix:///var/run/docker.sock")
//     if err != nil {
//       log.Fatalf("Error setting default DOCKER_HOST: %s", err)
//     }
//   }

//   newClient, err := client.NewEnvClient()
//   if err != nil {
//     log.Fatalf("Error creating docker client: %s", err)
//   }
//   docker = newClient

//   allImages, err := docker.ImageList(context.Background(), types.ImageListOptions{All: true})
//   if err != nil {
//     log.Fatalf("Error getting all docker images: %s", err)
//   }

//   imagesByID = make(map[string]types.Image, len(allImages))
//   for _, image := range allImages {
//     imagesByID[image.ID] = image
//   }
// }

func asSet(elements []string) map[string]struct{} {
	s := make(map[string]struct{}, len(elements))
	present := struct{}{}
	for _, e := range elements {
		s[e] = present
	}
	return s
}

// func ageOf(image types.Image) time.Duration {
//   return now.Sub(time.Unix(image.Created, 0))
// }

// func shortImageDigest(id string) string {
//   if strings.HasPrefix(id, "sha256:") {
//     return id[7:14]
//   }
//   return id
// }

// func cleanLeafImages() {
//   leafImages, err := docker.ImageList(context.Background(), types.ImageListOptions{})
//   if err != nil {
//     log.Fatalf("Error getting docker images: %s", err)
//   }

//   pruneContainerImages()

//   // Find images that are "root" images.
//   for _, image := range leafImages {
//     if imagesToSkip[image.ID] {
//       continue
//     }

//     for parentID := image.ParentID; len(parentID) != 0; parentID = imagesByID[parentID].ParentID {
//       image := imagesByID[parentID]
//       if len(image.RepoTags) == 1 && image.RepoTags[0] == "<none>:<none>" {
//         continue
//       }
//       if len(image.RepoDigests) == 1 && image.RepoDigests[0] == "<none>@<none>" {
//         continue
//       }

//       if !imagesToSkip[parentID] {
//         imagesToSkip[parentID] = true
//         log.Printf("Skipping tagged parent image %s: %s", shortImageDigest(parentID), strings.Join(image.RepoTags, ","))
//       }
//     }
//   }

//   pruneExcludedImages(leafImages)
//   pruneUnsafeImages(leafImages)

//   for _, image := range leafImages {
//     if imagesToSkip[image.ID] {
//       continue
//     }

//     nukeImage("leaf", image, *flagDeleteLeaf)
//   }
// }

// func cleanDanglingImages() {
// pruneContainerImages()
// pruneUnsafeImages(danglingImages)

// for _, image := range danglingImages {
//   if imagesToSkip[image.ID] {
//     continue
//   }

//   nukeImage("dangling", image, *flagDeleteDangling)
// }
// }

// func pruneContainerImages() {
//   containers, err := docker.ContainerList(context.Background(), types.ContainerListOptions{All: true})
//   if err != nil {
//     log.Fatalf("Error getting docker containers: %s", err)
//   }

//   // Find images belonging to containers.
//   for _, container := range containers {
//     inspected, err := docker.ContainerInspect(context.Background(), container.ID)
//     if err != nil {
//       log.Printf("Error getting container info for %s: %s", container.ID, err)
//       continue
//     }

//     for parent := inspected.Image; len(parent) != 0; parent = imagesByID[parent].ParentID {
//       if !imagesToSkip[parent] {
//         imagesToSkip[parent] = true
//         log.Printf("Skipping in use container image %s: %s", shortImageDigest(parent), strings.Join(imagesByID[parent].RepoTags, ","))
//       }
//     }
//   }
// }

// func pruneExcludedImages(images []types.Image) {
//   excluded := asSet(*flagExcludes)

//   for _, image := range images {
//     if imagesToSkip[image.ID] {
//       continue
//     }

//     for _, tag := range image.RepoTags {
//       if _, ok := excluded[tag]; ok {
//         log.Printf("Skipping excluded image %s: %s", shortImageDigest(image.ID), strings.Join(image.RepoTags, ","))
//         imagesToSkip[image.ID] = true
//       }
//     }
//   }
// }

// func pruneUnsafeImages(images []types.Image) {
//   for _, image := range images {
//     age := ageOf(image)
//     if age < *flagSafetyDuration {
//       if !imagesToSkip[image.ID] {
//         log.Printf("Skipping recent image %s: only %s old", shortImageDigest(image.ID), age)
//         imagesToSkip[image.ID] = true
//       }
//     }
//   }
// }

func nukeImage(kind string, image image.Image, reallyDelete bool) {
	if reallyDelete {
		fmt.Printf("Deleting %s image %s\n", kind, image)

		// var imagesToNuke []string
		// if len(image.RepoTags) <= 1 {
		//   imagesToNuke = append(imagesToNuke, image.ID)
		// } else {
		//   imagesToNuke = image.RepoTags
		// }
		// for _, imageIDOrTag := range imagesToNuke {
		//   _, err := docker.ImageRemove(context.Background(), types.ImageRemoveOptions{ImageID: imageIDOrTag, PruneChildren: true})
		//   if err != nil {
		//     log.Printf("Error while removing %s image %s: %s", kind, image.String(), err)
		//   }
		// }
	} else {
		fmt.Printf("Would have deleted %s image %s\n", kind, image)
	}
}
