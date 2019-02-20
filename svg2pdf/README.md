# svg2pdf

There is an existing implementatin of this component written in Python (https://github.com/Kozea/CairoSVG). This utility uses `cairo`, a C library, to *write* SVGs to a PDF. Interanally they have a self written xml and css parser which *reads* a source svg. These parsing libraries are also written in Python, but they aren't necessary with the advent of the `rsvg` C library. It has a function for reading SVG files into `cairo` surface objects. The pipeline for this is as follows:

1. create a `cairo` pdf surface object
2. create a `rsvg` "handle" backed by the source svg file
3. render the handle on the surface
4. safetly destruct the objects

The current implemntatin works as a command line utility and reads from a file to another file. The possibility of using buffers or OS level pipes needs to be explored.

## testfiles

The test files should be used for reference and not changed. The `.pdf`'s corresponding to the test `.svg`'s were included for visual tests.

## caveats

### file size
rsvg cannot handle files over 10MB, cursory research gives me the impression that it can be change with a build flag in `libxml`. This means the source svg files should *reference* an image rather than including vector paths or raw image.

The following is a list of the template files and their sizes. As such, the 5k template would not be renderable by rsvg. The `testfiles` dir contain one of each kind.

```
-rw-r--r--  1 @ear7h  staff   7.5M Dec 10 16:37 50k_template.svg
-rwxr-xr-x  1 @ear7h  staff    24M Dec 10 16:21 5k_template.svg
```

## static compile

Most of the libraries provide a static build (lib\*.a). The following do not:`gdk_pixbuf` and `librsvg`. I(@ear7h) was able to compile rsvg statically by using a depracted version (2.40.x) which supported a static build target. The newer versions are built partially (though they aim to move entirely) with rust and I am unfamiliar with its build systems. I did not look into gdk_pixbuf.

## resources
static compile:
* https://ubuntuforums.org/showthread.php?t=875883

svg2pdf:
* https://gitlab.com/cairo/cairo-demos/blob/master/PS/basket.c
* https://www.cairographics.org/manual/
* https://developer.gnome.org/rsvg/2.44/rsvg-Using-RSVG-with-cairo.html
