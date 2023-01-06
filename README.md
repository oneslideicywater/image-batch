# Image Batcher

`image-batch` export all images into single `.tar.gz`. you can load the images from `.tar.gz` in offline environments. 

the file generated by `image-batch` is usually smaller than files generated by `docker save xxx`. cuz it's backed by
docker registry which don't need to dump all images layer over and over again. 

## Usage
```bash
Usage:
  image-batch dump -f <filename> <tarfile>  dump all images in filename to tar.gz file
  image-batch load <tarfile>                load all images in the tar.gz file
```

## examples

```bash
$ cat imagelist 
busybox
nginx
# at online env
$ image-batch dump -f imagelist dump.tar.gz
...

# at offline env
$ image-batch load dump.tar.gz
```


## Reference List
- https://docs.docker.com/registry/