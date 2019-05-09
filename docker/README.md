# Docker

This directory contains Dockerfiles suitable for production and development.

# Dockerfile

This docker file will build a container sutiable for running the `atlante` program.

To build the conatiner, run the following command from the repo root:

```console
$ docker build -f docker/Dockerfile -t atlante .
```

To run the container from the root of the project.

```console
$ docker run --rm -v $(pwd):/mnt -it atlante
```

# Dockerfile-development

This docker file will build a container sutiable for development of the `atlante` program. Use the following command for building the container:

```console
$ docker build -f docker/Dockerfile-development -t atlante-dev .
```


Use this container in interactive mode with a volume mount so `atlante` can be built from source inside the container. For example, from the repository root:


```console
$ docker run --rm -v $(pwd):/go/src/github.com/go-spatial/maptoolkit -it atlante /bin/bash
``` 
