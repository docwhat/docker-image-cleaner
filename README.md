[![GitHub
release](https://img.shields.io/github/release/docwhat/docker-image-cleaner.svg)](https://github.com/docwhat/docker-image-cleaner/releases)
[![Build
Status](https://travis-ci.org/docwhat/docker-image-cleaner.svg?branch=master)](https://travis-ci.org/docwhat/docker-image-cleaner)

Docker image cleaner
====================

This command looks for images that should be okay to clean up.

This tool honors [semver](http://semver.org) versioning.

Usage
-----

    usage: docker-image-cleaner [<flags>]

    Clean up docker images that seem safe to remove.

    Flags:
      -h, --help                   Show context-sensitive help (also try --help-long
                                   and --help-man).
      -x, --exclude=IMAGE:TAG ...  Leaf images to exclude specified by image:tag
          --delete-dangling        Delete dangling images
          --delete-leaf            Delete leaf images
      -d, --safety-duration=DUR    Don't delete any images created in the last DUR
                                   time
          --version                Show application version.

It uses the normal Docker environment variables, so if `docker info` works,
then the cleaner should work.

### As a container

I have made this available as
[container](https://hub.docker.com/r/docwhat/image-cleaner/) as well.

    docker run --rm -v /var/run/docker.sock:/var/run/docker.sock docwhat/image-cleaner:latest

In addition to `:latest` it should also have all versions since 4.0.2

Credits
-------

-   This is based on [bobrik's
    docker-image-cleaner](https://github.com/bobrik/docker-image-cleaner).
    Thank you very much for sharing, @bobrik!
-   @seh provided lots of help for programming in Go (yeah, it's my first
    go program) and for what we should be cleaning up.
-   Also stole some travis tricks from taskcluster/slugid-go. Thanks for
    figuring that out!
