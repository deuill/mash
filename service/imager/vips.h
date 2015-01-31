#include <stdlib.h>
#include <vips/vips.h>
#include <vips/vips7compat.h>

int Vips_init() {
	return vips_init("alfred.imager.vips");
}

int Vips_load_jpeg(void *buf, size_t len, VipsImage **out) {
	return vips_jpegload_buffer(buf, len, out, "access", VIPS_ACCESS_SEQUENTIAL, NULL);
}

int Vips_load_png(void *buf, size_t len, VipsImage **out) {
	return vips_pngload_buffer(buf, len, out, "access", VIPS_ACCESS_SEQUENTIAL, NULL);
}

int Vips_save_jpeg(VipsImage *in, void **buf, size_t *len, int quality) {
	return vips_jpegsave_buffer(in, buf, len, "Q", quality, "strip", 1, "optimize_coding", TRUE, NULL);
}

int Vips_save_png(VipsImage *in, void **buf, size_t *len, int compression) {
	return vips_pngsave_buffer(in, buf, len, "compression", compression, NULL);
}

int Vips_shrink_load_jpeg(void *buf, size_t len, VipsImage **out, int shrink) {
	return vips_jpegload_buffer(buf, len, out, "shrink", shrink, NULL);
}

int Vips_shrink(VipsImage *in, VipsImage **out, double xshrink, double yshrink) {
	return vips_shrink(in, out, xshrink, yshrink, NULL);
};

int Vips_affine_bilinear(VipsImage *in, VipsImage **out, double a, double b, double c, double d) {
	VipsInterpolate *interpolator = vips_interpolate_new("bilinear");
	return vips_affine(in, out, a, b, c, d, "interpolate", interpolator, NULL);
};

int Vips_affine_bicubic(VipsImage *in, VipsImage **out, double a, double b, double c, double d) {
	VipsInterpolate *interpolator = vips_interpolate_new("bicubic");
	return vips_affine(in, out, a, b, c, d, "interpolate", interpolator, NULL);
};

int Vips_crop(VipsImage *in, VipsImage **out, int left, int top, int width, int height) {
	return vips_extract_area(in, out, left, top, width, height, NULL);
}

int Vips_colourspace(VipsImage *in, VipsImage **out, VipsInterpretation space) {
	return vips_colourspace(in, out, space, NULL);
};

int Vips_copy(VipsImage *in, VipsImage **out) {
	g_object_unref(*out);
	*out = vips_image_new();

	int result = vips_copy(in, out, NULL);
	g_object_unref(in);

	return result;
}