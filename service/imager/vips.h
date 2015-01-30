#include <stdlib.h>
#include <vips/vips.h>
#include <vips/vips7compat.h>

int vips_initialize() {
	return vips_init("alfred.imager.vips");
}

int vips_load_buffer_jpeg(void *buf, size_t len, VipsImage **out) {
	return vips_jpegload_buffer(buf, len, out, "access", VIPS_ACCESS_SEQUENTIAL, NULL);
}

int vips_load_buffer_png(void *buf, size_t len, VipsImage **out) {
	return vips_pngload_buffer(buf, len, out, "access", VIPS_ACCESS_SEQUENTIAL, NULL);
}