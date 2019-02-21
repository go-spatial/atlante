This directory contains patches to mapbox\_gl\_native library.

# Creating a patch

Since the patches are applied using `git apply` it's best to use
`git diff` to create the patches.

Example, given changes in file `a.go` and `b.go`
``` console

$ git diff a.go b.go > ${path_to_patches}/patches/000X-${description}

```
