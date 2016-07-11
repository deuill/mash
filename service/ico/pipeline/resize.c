#include <errno.h>
#include <math.h>
#include <stdlib.h>
#include <vips/vips.h>

#include "pipeline.h"
#include "resize.h"

void ico_image_shrink(ico_image *img, double factor) {
	// Return without shrinking if factor is less than 2.
	if (factor < 2) {
		errno = 0;
		return;
	}

	// JPEG images support a shrink-on-load operation, which is much more efficient
	// than generating a full-size image and shrinking afterwards.
	if (img->type == TYPE_JPEG) {
		int shrink = 2;
		VipsImage *tmp = NULL;

		if (factor >= 8) {
			shrink = 8;
		} else if (factor >= 4) {
			shrink = 4;
		}

		void *buf = (void *) img->data.buffer;
		size_t len = img->data.len;

		if (vips_jpegload_buffer(buf, len, &tmp, "shrink", shrink, NULL) != 0) {
			errno = 1;
			return;
		}

		g_object_unref(img->internal);
		img->internal = tmp;

		// Recalculate resize factor for shrunk image and return early if there
		// is no further processing required.
		factor = factor / shrink;
		if (factor < 2) {
			errno = 0;
			return;
		}
	}

	// Shrink image by integer factor.
	VipsImage *tmp = NULL;

	if (vips_shrink(img->internal, &tmp, floor(factor), floor(factor), NULL) != 0) {
		errno = 1;
		return;
	}

	g_object_unref(img->internal);
	img->internal = tmp;

	errno = 0;
	return;
}

void ico_image_affine(ico_image *img, double factor) {
	VipsImage *tmp = NULL;
	double residual = floor(factor) / factor;

	// Resize image by the residual factor. By default, uses a bilinear interpolator
	// for blending.
	if (vips_affine(img->internal, &tmp, residual, 0, 0, residual, NULL) != 0) {
		errno = 1;
		return;
	}

	g_object_unref(img->internal);
	img->internal = tmp;

	errno = 0;
	return;
}

void ico_image_crop(ico_image *img, int x, int y, int w, int h) {
	VipsImage *tmp = NULL;

	// Resize image by the residual factor. By default, uses a bilinear interpolator
	// for blending.
	if (vips_extract_area(img->internal, &tmp, x, y, w, h, NULL) != 0) {
		errno = 1;
		return;
	}

	g_object_unref(img->internal);
	img->internal = tmp;

	errno = 0;
	return;
}
