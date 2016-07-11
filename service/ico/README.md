# The Ico service

The Ico service for Mash provides methods for processing JPEG, PNG and GIF images, using S3 as a backing store. Images are processed against a pipeline, which is provided in the request, and which uniquely describes the resulting image in relation to the original image.

Ico service aims to be simple (both in use and in implementation), reliable and reasonably speedy, while allowing for deterministic results. Assuming the original image pointed to by the request is accessible and that the pipeline parameters are well-formed, Ico will always return a processed image, either from a local cache, the remote S3 store or by processing the image on-the-fly.

## Request structure

Assuming Mash is listening on an address `http://mash.deuill.org` and port `80`, a common GET request would be in this form:

```
http://mash.deuill.org/ico/width=500,fit=crop/header/promo/kittens-hats.jpg
<--------- 1 -------->< 2 ><------ 3 -------><----------- 4 -------------->
```

The request URL contains 4 distinct parts:

  1. The hostname on which Mash is listening, and which is used for accessing all services attached to the Mash instance.
  2. The service name, which is unique to each service.
  3. The pipeline parameters, describing the resulting image.
  4. The original image URL, relative to the S3 bucket root directory.

A request of this form would first attempt to fetch the processed image from the local and remote cache, and failing that, would create the image on-the-fly, populate the caches for the benefit of any future requests, and return the processed image to the user.

## Image processing

Image processing is handled via [VIPS](http://www.vips.ecs.soton.ac.uk), which is compiled into the Ico service as a C library. VIPS was chosen due to its excellent [performance characteristics](http://www.vips.ecs.soton.ac.uk/index.php?title=Speed_and_Memory_Use), its stability, and its clean and simple API.

More information on the image processing pipeline can be found in the [README file](https://github.com/deuill/mash/blob/master/service/ico/pipeline/README.md) for the pipeline package.

## Image caching

Ico caches processed images in both a local cache and a remote cache, using S3. A rationale and description of caching strategies for each component is described below.

### Local cache

The local cache operates under the principles of an LRU-type algoarithm. A disk quota is set aside for cache (can be unlimited), and items are placed in a doubly-linked list. Whenever an item is added or accessed, it is moved to the front of the list. When attempting to add an item that would cause the cache size to exceed its alloted quota, items are removed from the end of the list until the size requirements are satisfied.

Though accessing files on S3 is reasonably quick, the time between a processed image being generated and that image being uploaded to S3 can mean identical requests have to wait, when a local cache would allow such requests to return immediately.

### S3 cache

Processed images are uploaded back to the same S3 bucket and directory hosting the original file, following a naming scheme consistent with the request presented in the URL. For the above example, the full path for the resulting image would be `/header/promo/width=500,fit=crop/kittens-hats.jpg`.

Thus, processed images are stored in a directory named after the pipeline parameters that were used for generating them, under the same directory as their originals. This makes it possible to reconstruct the URL parameters used for generating the image stored in a reverse manner. It also allows applications with no knowledge of Ico's internal workings, i.e. a CDN, to fetch images directly from S3 using the same URL request structure as what would be passed Ico.

## Configuration

Ico conforms to the Mash standard of requiring the least amount of configuration state possible for functional use. Since all information required for processing images is passed in the request, the only remaining state pertains to the cache quota and any details required for S3 access, such as region name, bucket name, access key and secret key.

However, since Ico allows for the region and bucket names to be provided in the `X-S3-Region` and `X-S3-Bucket` request headers, and, assuming access to S3 is provided via IAM for the running server, most configuration state is optional, and is mainly useful for small deployments or development.
