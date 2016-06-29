# The Ico service

The Ico service for Mash provides methods for processing JPEG, PNG and GIF images, using S3 as a backing store. Images are processed against a pipeline, which is provided in the request, and which uniquely describes the resulting image in relation to the original image.

Ico service aims to be simple (both in use and in implementation), reliable and reasonably speedy, while allowing for deterministic results. Assuming the original image pointed to by the request is accessible and that the pipeline parameters are well-formed, Ico will always return a processed image, either from a local cache, the remote S3 store or by processing the image on-the-fly.

## Request structure

Assuming Mash is listening on an address `http://mash.deuill.org` and port `80`, a common GET request would be in this form:

```
http://mash..deuill.org/ico/d2lkdGg9NTAwLGZpdD1jcm9wCg==/header/promo/kittens-hats.jpg
<--------- 1 --------->< 2 ><----------- 3 ------------><----------- 4 -------------->
```

The request URL contains 4 distinct parts:

  1. The hostname on which Mash is listening, and which is used for accessing all services attached to the Mash instance.
  2. The service name, which is unique to each service.
  3. The base64-encoded pipeline parameters (in this case, `width=500,fit=crop`).
  4. The original image URL, relative to the S3 bucket root directory.

A request of this form would first attempt to fetch the processed image from the local and remote cache, and failing that, would create the image on-the-fly, populate the caches for the benefit of any future requests, and return the processed image to the user.

While passing the pipeline parameters as base64-encoded strings in the URL may appear unorthodox, it allows the service to be used directly with no need for an SDK or any complex request header manipulation. It makes debugging easier, since a URL is also a complete description of the resulting image. Finally, it allows for orthogonal URL structures, which may become apparent in the caching
strategy, described further below.

## Image processing

Image processing is handled via [VIPS](http://www.vips.ecs.soton.ac.uk), which is compiled into the Ico service as a C library. VIPS was chosen due to its excellent [performance characteristics](http://www.vips.ecs.soton.ac.uk/index.php?title=Speed_and_Memory_Use), its stability, and its clean and simple API.

The pipeline parameters provided with the request describe the resulting image, and conform to the following specification:

Parameter name | Description                                         | Accepted values                 | Default value
---------------|-----------------------------------------------------|---------------------------------|--------------
width          | Image width. If 0, calculate from height            | 0 ... infinity                  | 0
height         | Image height. If 0, calculate from width            | 0 ... infinity                  | 0
quality        | Image quality for JPEG and compression rate for PNG | 1 ... 100                       | 75
fit            | Fit mode. If "clip", resize without cropping        | clip, crop                      | crop
crop           | Cropping strategy                                   | top, bottom, left, right, focus | top
focus          | Bounding box for "focus" strategy                   | width:height:x-pos:y-pos        | 0:0:0:0
frame          | If "true", only returns first frame of GIF          | true, false                     | false

Parameters are comma-separated key-value assignments, for example `width=500,fit=crop`. Certain parameters have additional constraints on their values, as described below:

### `width` and `height`

These parameters accept any integer value, but negative numbers and values that are equal or exceed the original image's resolution result in the original image being returned.

### `quality`

This is an integer between 1 and 100, and affects the quality of JPEG images as expected. For PNG images, the value is scaled between 0 and 9, and is used to determine the compression rate for the resulting image.

### `fit`

Determines whether an image should be cropped when resizing. For example, for an image size of **500x1000** and a pipeline of `width=400,height=400,fit=...`, `fit=clip` would result in an image size of **200x400**, whereas `fit=crop` would result in an image size of **400x400**, with extra pixels being removed according to the cropping strategy.

### `crop`

Determines the cropping strategy. Values `top`, `bottom`, `left` and `right` determine which part of the image is cut off. In the above example, the default cropping strategy of `top` would result in the top 200 pixels being removed. Strategy `focus` defines that the center of gravity for the crop is determined by a bounding box, as passed in the `focus` parameter.

### `focus`

This parameter defines the center of gravity for image crops as a bounding box. The bounding box is defined as four, colon-separated, integer values corresponding to the width, height, X and Y position of the bounding box relative to the original image's dimensions.

### `frame`

If `true`, and we're processing an animated GIF file, this will only process and return the first frame in the image.

## Image caching

Ico caches processed images in both a local cache and a remote cache, using S3. A rationale and description of caching strategies for each component is described below.

### Local cache

The local cache operates under the principles of an LRU-type algoarithm. A disk quota is set aside for cache (can be unlimited), and items are placed in a doubly-linked list. Whenever an item is added or accessed, it is moved to the front of the list. When attempting to add an item that would cause the cache size to exceed its alloted quota, items are removed from the end of the list until the size requirements are satisfied.

Though accessing files on S3 is reasonably quick, the time between a processed image being generated and that image being uploaded to S3 can mean identical requests have to wait, when a local cache would allow such requests to return immediately.

### S3 cache

Processed images are uploaded back to the same S3 bucket and directory hosting the original file, following a naming scheme consistent with the request presented in the URL. For the above example, the full path for the resulting image would be `/header/promo/d2lkdGg9NTAwLGZpdD1jcm9wCg==/kittens-hats.jpg`.

Thus, processed images are stored in a directory named after the pipeline parameters that were used for generating them, under the same directory as their originals. This makes it possible to reconstruct the URL parameters used for generating the image stored in a reverse manner. It also allows applications with no knowledge of Ico's internal workings, i.e. a CDN, to fetch images directly from S3 using the same URL request structure as what would be passed Ico.

## Configuration

Ico conforms to the Mash standard of requiring the least amount of configuration state possible for functional use. Since all information required for processing images is passed in the request, the only remaining state pertains to the cache quota and any details required for S3 access, such as region name, bucket name, access key and secret key.

However, since Ico allows for the region and bucket names to be provided in the `X-S3-Region` and `X-S3-Bucket` request headers, and, assuming access to S3 is provided via IAM for the running server, most configuration state is optional, and is mainly useful for small deployments or development.
