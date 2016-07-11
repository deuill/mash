#include <errno.h>
#include <stdlib.h>
#include <vips/vips.h>

#include "pipeline.h"

int ico_init() {
	if (vips_init("mash.ico.vips") != 0) {
		vips_shutdown();
		return 1;
	}

	vips_concurrency_set(1);
	vips_cache_set_max_mem(1024 * 1024 * 128); // 128MB
	vips_cache_set_max(256);                   // 256 operations

	return 0;
}

ico_image *ico_image_new(const void *data, size_t len, int type) {
	ico_image *img;

	// Allocate initial image structure.
	img = malloc(sizeof(ico_image));
	if (img == NULL) {
		errno = 1;
		return NULL;
	}

	// Attempt to load internal representation of image from buffer via VIPS.
	img->internal = vips_image_new_from_buffer(data, len, "", NULL);
	if (img->internal == NULL) {
		errno = 1;
		return NULL;
	}

	img->data.buffer = data;
	img->data.len = len;
	img->type = type;

	errno = 0;
	return img;
}

void ico_image_write(ico_image *img, void **buf, size_t *len) {
	int result;

	// Determine image type to write.
	switch (img->type) {
	case TYPE_JPEG:
		result = vips_jpegsave_buffer(img->internal, buf, len, NULL);
		break;
	case TYPE_PNG:
		result = vips_pngsave_buffer(img->internal, buf, len, NULL);
		break;
	case TYPE_GIF:
		// Saving to GIF not supported yet.
		errno = 1;
		return;
	}

	// Check for possible error during processing.
	if (result != 0) {
		errno = 1;
		return;
	}

	errno = 0;
	return;
}

void ico_image_destroy(ico_image *img) {
	g_object_unref(img->internal);
	free(img);
}

int ico_image_width(ico_image *img) {
	return vips_image_get_width(img->internal);
}

int ico_image_height(ico_image *img) {
	return vips_image_get_height(img->internal);
}
