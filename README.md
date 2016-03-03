[![GitHub
release](https://img.shields.io/github/release/docwhat/docker-image-cleaner.svg)](https://github.com/docwhat/docker-image-cleaner/releases)
[![Build
Status](https://travis-ci.org/docwhat/docker-image-cleaner.svg?branch=master)](https://travis-ci.org/docwhat/docker-image-cleaner)

Docker image cleaner
====================

This command looks for images that should be okay to clean up.

Usage
-----

Add `-dry-run` to the end if you want to see what is going to be deleted.

If you want to keep some images, use `-exclude image:tag[,image:tag]` flag.

It uses the normal Docker environment variables, so if `docker info` works,
then the cleaner should work.

Credits
-------

This is based on [bobrik's
docker-image-cleaner](https://github.com/bobrik/docker-image-cleaner). Thank
you very much for sharing, bobrik!
