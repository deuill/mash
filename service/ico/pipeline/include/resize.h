#ifndef __RESIZE_H__
#define __RESIZE_H__

void ico_image_shrink(ico_image *img, double factor);
void ico_image_affine(ico_image *img, double factor);
void ico_image_crop(ico_image *img, int x, int y, int w, int h);

#endif
