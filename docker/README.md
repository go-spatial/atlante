# Docker

This directory contains Dockerfiles suitable for production and development.

## Docker Hub

`atlante` is hosted on Docker Hub at https://hub.docker.com/r/gospatial/atlante. Here's an example of pulling the container and running it locally:

``` console
docker pull gospatial/atlante

# Command Line Interface help
docker run --rm -it gospatial/atlante -h

# run with the current directory mounted
docker run --rm -v "$(pwd)":/mnt -it gospatial/atlante --help

# Mount local directory as `/mnt`, using `./config.toml`
docker run --rm -v "$(pwd)":/mnt -it gospatial/atlante --config /mnt/config.toml

# test other parameters
docker run --rm -v "$(pwd)":/mnt -it gospatial/atlante \
  --config /mnt/config.toml                            \
  --dpi 144 --sheet sheetIndex -o /mnt
```

### Example `config.toml`

``` toml
[[providers]]
  name     = "providerName"
  type     = "postgresql"            # Types are defined at atlante⁩/⁨atlante/⁨grids
  host     = "host.docker.internal"  # Required for macOS, https://docs.docker.com/docker-for-mac/networking
  # host   = "192.168.1.100"         # You can use IP address or domain (postgis1.example.com)
  port     = 5432                    # postgresql database port
  database = "bonn"
  user     = "tegola"
  password = ""
```

# Dockerfile-atlante

This [docker file](docker/Dockerfile-atlante) will build a container suitable for running the `atlante` program.

To build the container, run the following command from the repo root:

```console
$ docker build -f docker/Dockerfile-atlante -t atlante .
```

To run the container from the root of the project.

```console
$ docker run --rm -v $(pwd):/mnt -it atlante
```

# Dockerfile-atlante-development

This docker file will build a container sutiable for development of the `atlante` program. Use the following command for building the container:

```console
$ docker build -f docker/Dockerfile-atlante-development -t atlante-dev .
```

Use this container in interactive mode with a volume mount so `atlante` can be built from source inside the container. For example, from the repository root:

```console
$ docker run --rm -v $(pwd):/go/src/github.com/go-spatial/atlante -it atlante-dev /bin/bash
```

# Dockerfile-atlante-inset-maps

This [docker file](docker/Dockerfile-atlante-inset-maps) will build a container sutiable for running the `atlante-inset-maps` server. Use the following command for building the container:

```console
$ docker build -f docker/Dockerfile-atlante-inset-maps -t atlante-inset-maps .
```

To run the container from the root of the project.

```console
$ HTTP_PORT=8080 docker run --rm -p8080:8080 -v $(pwd):/mnt -t atlante-inset-maps --config /mnt/config.toml
```

Note: the config should point to things inside of the container. The aboved assumes that config has the port set to `${HTTP_PORT}`
