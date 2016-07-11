# The Image Processing Pipeline

An image processing pipeline is the representation of an ordered list of operations, and is a complete description of the processed result for any particular original image.

Pipelines are identified by the parameters that describe them, for example, a parameter list `saturation=10,width=500` will correspond to the following pipeline:

```
Pipeline {
	Resize {
		Width: 500,
	},
	Adjust {
		Saturation: 10,
	},
}
```

The above is a complete description of the manipulations required for processing an image. Also, note that the order of parameters does not affect the final order of operations, and thus a pipeline can be identified by more than one parameter lists.

Parameters are comma-separated key-value assignments, for example `width=500,fit=crop`. Certain parameters have additional constraints on their values, as described below.

## Operations

Operations are the building blocks of the image processing pipeline, and are defined as sets of related image manipulation tasks, e.g. resizing, adjusting colors etc.

What follows is a reference list of all available operations, along with a list of parameters relevant to each one.

### Resize

The resize operation handles any manipulation of the image's dimensions, including clipping and cropping. The parameters relevant to this operation are:

Name   | Description                              | Accepted Values | Default Value
-------|------------------------------------------|-----------------|--------------
width  | Image width. If 0, calculate from height | 0 ... infinity  | 0
height | Image height. If 0, calculate from width | 0 ... infinity  | 0
fit    | Fit mode for resized image               | crop            | clip


#### `width` and `height`

These parameters accept any integer value, but negative numbers and values that are equal or exceed the original image's resolution result in the original image being returned.

#### `fit`

Determines the way in which the image will attempt fit the constraints imposed by the pipeline. Supported fit modes and their additional options include:

  * `clip`: Attempts to resize image so that resulting image dimensions are smaller or equal to the pipeline constraints. So, for an image of size `1000x500` and a pipeline of `width=500,height=200`, the resulting image will be of size `400x200`. This is the default.
  * `crop`: Attempts resize image to the exact size requested, cropping any additional parts of the image. Supports the following colon-separated options:
    * `top`, `bottom`, `left`, `right`, `center`, which define the center of gravity for the cropped image. So, for the above example and a fit of `fit=crop:bottom`, the top 50 pixels of the image would be cropped. Default is `center`.
	* `point`, which defines the center of gravity for a cropped image as X and Y pixel co-ordinates. For example, the center point of focus for the above example would be expressed by a pipeline of `fit=crop:point:500:250`.
