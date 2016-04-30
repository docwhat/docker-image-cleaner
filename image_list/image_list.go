package image_list

import "docwhat.org/docker-image-cleaner/image"

type Interface interface {
}

type ImageList struct {
	images []image.Interface
}

type filter func(image.Interface) bool

func (source *ImageList) Select(f filter) ImageList {
	list := make([]image.Interface, len(source.images))

	for _, img := range source.images {
		if f(img) {
			list = append(list, img)
		}
	}

	return ImageList{images: list}
}

func (source *ImageList) Orphans() ImageList {
	filter := func(i image.Interface) bool { return i.IsOrphan() }
	return source.Select(filter)
}
