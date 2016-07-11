#ifndef __PIPELINE_H__
#define __PIPELINE_H__

typedef struct __ico_image {
	VipsImage *internal;
	struct {
		const void *buffer;
		size_t len;
	} data;
	int type;
} ico_image;

enum {
	TYPE_JPEG,
	TYPE_PNG,
	TYPE_GIF,
};

int ico_init();

ico_image *ico_image_new(const void *data, size_t len, int type);
void ico_image_write(ico_image *img, void **buf, size_t *len);
void ico_image_destroy(ico_image *img);

int ico_image_width(ico_image *img);
int ico_image_height(ico_image *img);

#endif
