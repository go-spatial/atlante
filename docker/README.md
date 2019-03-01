# Docker

this directory contains Dockerfiles for various environments. To build them
use the `-f` flag like so:

Fonts placed into the _file/fonts_ folter will be copied to the _/usr/share/fonts/_ directory.


Run this from the root of the project.
```console

 $ docker build -f docker/Dockerfile.centos -t maptoolkit .

```

To run the container from the root of the project.

```console

 $ docker run --rm -v "$(pwd)":/go/src/github.com/go-spatial/maptoolkit -it maptoolkit

``` 


# Dockerfile.centos
This docker file will build a container sutiable for running `atlante` program.

# Dockerfile-development.centos
This docker file will build a container sutiable for development of the `atlante` program.

