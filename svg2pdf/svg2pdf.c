#include <stdlib.h>
#include <glib/gstdio.h>
#include <librsvg/rsvg.h>
#include <cairo-pdf.h>
#include <cairo-ps.h>

int svg2pdf_file(const char * inFile, const char * outFile,
		double height, double width) {
	cairo_t * cr;
	cairo_surface_t * surface;
	RsvgHandle * handle;
	GError * error = NULL;
	GFile * file;
	GInputStream * stream;
	RsvgHandleFlags flags;

#if DEBUG 

	printf("%s\n", "start");

	printf("%p\n", cairo_pdf_surface_create);
	printf("%d\n", CAIRO_HAS_PDF_SURFACE);

	printf("creating handle to %s\n", inFile);

#endif

	file = g_file_new_for_path(inFile);
	stream = (GInputStream *) g_file_read(file, NULL, &error);
	if (stream == NULL){

#if DEBUG

		printf("%s\n", "could not create read stream");

#endif
		return 1;
	}

	flags |= RSVG_HANDLE_FLAG_UNLIMITED;

	handle = rsvg_handle_new_from_stream_sync(stream, file, flags, NULL,
		&error);
	if (handle == NULL) {

#if DEBUG
		printf("error (%d) %s\n", error->code, error->message);
#endif

		return 1;
	}

#if DEBUG
	printf("%p\n", handle);
#endif

	// set handle options

	surface = cairo_pdf_surface_create(outFile, height, width);
	cairo_status_t status = cairo_surface_status(surface);
	if (status != CAIRO_STATUS_SUCCESS) {
#if DEBUG
		printf("%s\n", cairo_status_to_string(status));
#endif
		return 1;
	}
#if DEBUG
	printf("surface created %p\n", ((void *) surface));
#endif

	cr = cairo_create(surface);
	status = cairo_status(cr);
	if (status != CAIRO_STATUS_SUCCESS) {
#if DEBUG
		printf("%s\n", cairo_status_to_string(status));
#endif
		return 1;
	}
#if DEBUG
	printf("context created %p\n", ((void *) cr));
#endif

	gboolean ok;
	ok = rsvg_handle_render_cairo(handle, cr);
#if DEBUG
	printf("render returned\n");
#endif
	if (!ok) {
#if DEBUG
		printf("error\n");
#endif
		return 1;
	}

	// TODO(gdey): is this needed?
	cairo_surface_write_to_png(surface, "example.png");
#if DEBUG
	printf("handle rendered\n");
#endif

	cairo_surface_destroy(surface);
	cairo_destroy(cr);
	rsvg_handle_close(handle, &error);

	return 0;
}
