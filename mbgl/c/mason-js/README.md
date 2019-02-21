`circleCI` [![CircleCI](https://circleci.com/gh/mapbox/mason-js.svg?style=svg)](https://circleci.com/gh/mapbox/mason-js)

[![codecov](https://codecov.io/gh/mapbox/mason-js/branch/master/graph/badge.svg)](https://codecov.io/gh/mapbox/mason-js)

# mason-js: a javascript client for mason 

## Why mason-js? 

Mason-js is a JS client for [Mason, the C++ package manager.](https://github.com/mapbox/mason) 

This project is: 

- The first standalone cross-platform client for Mason.
- Makes installing packages seamless for node c++ addons.
- Could also be used in stacks that have nodejs as a dep. 
- We are able to remove mason’s custom install scripts and house all of our logic in a JS client. [Diff of install script vs mason-js in node-cpp-skel.](https://github.com/mapbox/node-cpp-skel/compare/proposed-mason-js-port)

## How to use this? 

**Install all package dependencies** 

```mason-js install```  
- installs all packages from the `mason-versions.ini` file 

Example `mason-versions.ini`:

  
    [headers]
    protozero=1.6.1
    boost=1.65.1
    [compiled]
    jpeg_turbo=1.5.1
    libpng=1.6.28
    libtiff=4.0.7
    icu=57.1
    proj=4.9.3
    pixman=0.34.0
    cairo=1.14.8

**Symlink Installed Packages**

```mason-js link``` 
- symlinks packages 

**Install a Single Package**

```mason-js <package>=<version> --type=[header or compiled]```  
- installs a single package  

*Example*

`mason-js protozero=1.5.1 --type=header`

## What is V1? 
- Ability to `install`  all packages from a mason-versions.ini file 
- Ability to `link` all packages from a mason-versions.ini file 
- Ability to install a single package 
- Available on Mac, Linux, and Windows environments 
  - Tests via CircleCI & Apprevyor 

In short, mason-js: 

1. **Reads** a `mason-versions.ini` file in the [root](https://github.com/mapbox/node-cpp-skel/compare/proposed-mason-js-port#diff-8a4b16fa77ffdd0d617a663440578a2d) [directory](https://github.com/mapbox/node-cpp-skel/compare/proposed-mason-js-port#diff-8a4b16fa77ffdd0d617a663440578a2d) [of a project](https://github.com/mapbox/node-cpp-skel/compare/proposed-mason-js-port#diff-8a4b16fa77ffdd0d617a663440578a2d).
2. **Installs** header only and precompiled packages from the S3 bucket `mason-binaries` into a `mason-packages` folder 
3. **Creates symlinks** a `.link` directory to package executables  


**Remote** **Package Paths****:** 

- Header-only: `s3://mason-binaries/headers/{package}/{version}.tar.gz`

- Binaries: `s3://mason-binaries/{MASON_PLATFORM}-{MASON_PLATFORM_VERSION}/{package}/{version}.tar.gz`


Note: The value of `MASON_PLATFORM_VERSION` is determined per platform. 

**Local** **Package Paths**:


- Header-only: `./mason_packages/headers/{package}/{version}/<source files>`
  - e.g. `mason_packages/headers/vtzero/556fac5/include/vtzero/vector_tile.hpp`


- Binaries: `./mason_packages/{platform}-{arch}/{package}/{version}/<files>`
  - e.g. `mason_packages/osx-x86_64/gdal/2.2.1/bin/ogr2ogr`


- Linked folder: `./mason_packages/.link/<files>`
  - e.g. `mason_packages/.link/lib/libpng.a`


# Development 

**Install Local Dependencies** 

```npm install```

```npm link```

**Run Tests**

```npm test``` 

**Make commands**

The Make file has a series of commands that run a Docker container locally. By using these commands, you’re able to simulate staging/production environments locally.

Make sure to set your `NPMAccessToken` in your environment before running any `make` commands.


- `make bash` - opens a terminal shell session in your Docker image
- `make build` - build your Docker image locally
- `make run` - send a message to the queue. (this command runs make build first)
- `make test` - run tests



