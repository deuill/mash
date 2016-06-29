#include <stdlib.h>
#include <vips/vips.h>
#include <vips/vips7compat.h>

int Vips_init() {
	return vips_init("mash.ico.vips");
}

VipsImage *Vips_image_init() {
	VipsImage *image = NULL;
	return image;
}

VipsImage *Vips_load_jpeg(void *buf, size_t len) {
	VipsImage *image = NULL;
	vips_jpegload_buffer(buf, len, &image, "access", VIPS_ACCESS_SEQUENTIAL, NULL);
	return image;
}

VipsImage *Vips_shrink_load_jpeg(void *buf, size_t len, int shrink) {
	VipsImage *image = NULL;
	vips_jpegload_buffer(buf, len, &image, "shrink", shrink, NULL);
	return image;
}

void *Vips_save_jpeg(VipsImage *in, size_t *len, int quality) {
	void *buf = NULL;
	vips_jpegsave_buffer(in, &buf, len, "Q", quality, "strip", 1, "optimize_coding", TRUE, NULL);
	return buf;
}

VipsImage *Vips_load_png(void *buf, size_t len) {
	VipsImage *image = NULL;
	vips_pngload_buffer(buf, len, &image, "access", VIPS_ACCESS_SEQUENTIAL, NULL);
	return image;
}

void *Vips_save_png(VipsImage *in, size_t *len, int compression) {
	void *buf = NULL;
	vips_pngsave_buffer(in, &buf, len, "compression", compression, NULL);
	return buf;
}

VipsImage *Vips_shrink(VipsImage *in, double xshrink, double yshrink) {
	VipsImage *image = NULL;
	vips_shrink(in, &image, xshrink, yshrink, NULL);
	return image;
};

VipsImage *Vips_affine_bilinear(VipsImage *in, double a, double b, double c, double d) {
	VipsImage *image = NULL;
	VipsInterpolate *interpolator = vips_interpolate_new("bilinear");

	vips_affine(in, &image, a, b, c, d, "interpolate", interpolator, NULL);
	g_object_unref(interpolator);

	return image;
};

VipsImage *Vips_affine_bicubic(VipsImage *in, double a, double b, double c, double d) {
	VipsImage *image = NULL;
	VipsInterpolate *interpolator = vips_interpolate_new("bicubic");

	vips_affine(in, &image, a, b, c, d, "interpolate", interpolator, NULL);
	g_object_unref(interpolator);

	return image;
};

VipsImage *Vips_crop(VipsImage *in, int left, int top, int width, int height) {
	VipsImage *image = NULL;
	vips_extract_area(in, &image, left, top, width, height, NULL);
	return image;
}

VipsImage *Vips_colourspace(VipsImage *in, VipsInterpretation space) {
	VipsImage *image = NULL;
	vips_colourspace(in, &image, space, NULL);
	return image;
};
