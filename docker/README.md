# Docker

this directory contains Dockerfiles for various environments. To build them
use the `-f` flag like so:


Run this from the root of the project.
```console

 $ docker build -f docker/Dockerfile.debian -t maptoolkit .

```

To run the container from the root of the project.

```console

 $ docker run -it --rm -v "$(pwd)":/go/src/github.com/go-spatial/maptoolkit

```

