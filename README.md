# Artefactor

[![Build_Status](https://circleci.com/gh/appvia/artefactor.svg?style=svg)](https://circleci.com/gh/appvia/artefactor)

Artefactor's primary use case is to enable "moving" a git repository and the
 artefacts it depends on when using a 
 [Sneakernet](https://en.wikipedia.org/wiki/Sneakernet).

- [Details](#details)
- [Usage](#usage)
  - [Use in CI/CD](#usage-in-ci)
- [Build](#build)
- [Roadmap](#roadmap)

## Details

E.g. a git repo for a [kubernetes](https://kubernetes.io/) deployment is 
specified in a git repo that depends on:

- The git repository files (kubernetes yaml)
- docker images

Although simple to describe, the steps can be problematic if suitable steps are
not taken to:

- verify artefacts (checksums)
- restore files back on remote systems
  - load / publish container images
  - set executable bit for tools / restore archives
- support safe file names for removable media across multiple platforms
- cache large downloads

## Usage

### save

`artefactor save` will save all artefacts as specified by flags / environment 
e.g. the example below archives the current git repo, a container and a 
binary from a github release:

Artefacts can be specified as flags or as environment variables with the following formats:

| flag      | format | description | example |
|-----------|--------|-------------|---------|
| `--git-repos` | [.] [local path] | Will archive a git repository. If the directory is the same as ${PWD}, it signifies the "home" for restoring. | `.` |
| `--docker-images` | docker-image docker-image | A white-space delimited set of docker images | `mysql alpine` |
| `--image-vars` | `"MYSQL_IMAGE ANOTHER_IMAGE"` | A white-space delimited set of image variable names | Given:</br>`export MYSQL_IMAGE=mysql:v5.0`</br>`export ALPINE_IMAGE=alpine` </br> Use: </br>`"MYSQL_IMAGE ALPINE_IMAGE"`|
| `--web-files` | url,filename,sha256[,true/false] | A white-space separated list of CSV's in the following format: </br></br>`url` is where to download from</br></br> `filename` is the name to save locally</br></br> `sha256` is the expected checksum</br></br>The optional last parameter specifies if the file should have executable permissions | `https://bit.ly/2ySXztI,kd,2f7...,true https://bit.ly/abc.iso,my.iso,abc...` |
| `--docker-username` | `username` | A valid docker registry user-name see # | `bob` |
| `--docker-password` | `testing` | A valid docker registry password | `testing` |

*Common Flags:*

```bash
artefactor save --git-repos=.\
                --docker-images="mysql alpine" \
                --web-files https://github.com/UKHomeOffice/kd/releases/download/v0.13.0/kd_linux_amd64,kd,2f729bb26e225bcf61aa62a03d210f9a238d1c7b1666c1d72964decf7120466a,true
```

*Environment:*

```bash
export ARTEFACTOR_DOCKER_IMAGES="mysql
                                 alpine:latest"
export ARTEFACTOR_WEB_FILES="https://github.com/UKHomeOffice/kd/releases/download/v0.13.0/kd_linux_amd64,kd,2f729bb26e225bcf61aa62a03d210f9a238d1c7b1666c1d72964decf7120466a,true"

artefactor save
```

*Environment Image Lists:*

It is useful to be able to specify a list of variables used for
images. In this case, the flag `--image-vars`
(`ARTEFACTOR_IMAGE_VARS`) can be specified directly.

```bash
MYSQL_IMAGE=mysql:v5.0
CASSANDRA_IMAGE=docker.io/cassandra:latest
ARTEFACTOR_IMAGE_VARS="MYSQL_IMAGE CASSANDRA_IMAGE"
ARTEFACTOR_DOCKER_REGISTRY=myreg.local

artefactor save
```

### restore

`artefactor restore` will restore artefacts to the original layout.
 (e.g. repo and ./downloads by default). It uses the meta data files stored with
 saved files to restore file permissions and structure.

*Common Flags:*

To restore a set of artefactor saved files from a home directory to the current
working directory:

```bash
artefactor restore --source-dir ~/
```

### publish

`artefactor publish` takes files from the relative ./downloads path and 
publishes containers / files to any remote registries / locations.

### update-image-vars

Artefactor can update environment variables with a list of transformed image
names. This is useful when environment variables with image names are used in
deployments and they need to be managed with a private registry name.

Given the following exported shell variables:

```bash
export MYSQL_IMAGE=mysql:v5.0
export CASSANDRA_IMAGE=docker.io/cassandra:latest
export ARTEFACTOR_IMAGE_VARS="MYSQL_IMAGE CASSANDRA_IMAGE"
export ARTEFACTOR_DOCKER_REGISTRY=myreg.local
```

To safely update environment variables with a list of images (as discovered from the
 downloaded artefacts):

```bash
set -e
exports=$(artefactor update-image-vars)
eval "echo ${exports}"
```

The sub command will produce the output:

```bash
export MYSQL_IMAGE=myreg.local/mysql:v5.0
export CASSANDRA_IMAGE=myreg.local/cassandra:latest
```

**Note**: only images known to artefactor (from the downloads meta-data) and
 variable names white-listed using `ARTEFACTOR_IMAGE_VARS` or the flag
`--image-vars` will "be exported".

#### Content Adressable Repo Digests

When updating image vars and addressing images using the digest format, eg. `alpine@sha256:6a92cd1fcdc8d8cdec60f33dda4db2cb1fcdcacf3410a8e05b3741f44a9b5998`
The `repoDigest` cannot be guaranteed to be the same between registries due to differences in implementations of digest calculation. For this reason, when updating image vars, a local docker instance must have already published the image being updated so a new repoDigest is available and can be updated in the image var.
for example:

```bash
export ALPINE_IMAGE="alpine@sha256:6a92cd1fcdc8d8cdec60f33dda4db2cb1fcdcacf3410a8e05b3741f44a9b5998"
export ARTEFACTOR_IMAGE_VARS="ALPINE_IMAGE"
export ARTEFACTOR_DOCKER_REGISTRY=myreg.local
set -e 
exports=$(artefactor update-image-vars)
eval "echo ${exports}"
```

the resulting sub command will produce the output similar to:

```bash
export MYSQL_IMAGE=myreg.local/alpine@sha256:b7b28af77ffec6054d13378df4fdf02725830086c7444d9c278af25312aa39b9
```

As can be seen the digest sha has changed. This was picked up from the metadata stored in the local docker instance gathered from the push even `artefactor publish` generates.

**Notes**:

- if the image has not been published from the context the `artefactor update-image-vars` command is being run from, the command will fail with an error not being able to find the image details in the local docker instance.
- Image stored with the format `imagename:vX.Y.Z@sha256:[64chars]` will be loaded with only an image name and no tag, and then tagged twice when pushed to the destination registry, both with the original tag AND a second reference to the repoDigest it was pulled from at the source registry. This is a convenience to provide a reverse reference for the source of the image since there is no direct reference possible between separate registries.

### Usage in CI

In CI/CD environments the following docker authentication environment variables
and corresponding flags are supported for both the `save` and `publish` sub commands:

```bash
export ARTEFACTOR_DOCKER_USERNAME=bob
export ARTEFACTOR_DOCKER_PASSWORD=testing
```

**Note**: A local docker daemon is required for publishing containers to 
registries but not for saving from registries.

## Build

Binaries are created in `./bin/`.

To install dependencies and build:
`make`

To Build with dependencies and test:
`make test`

To build quickly:
`make build`

## Roadmap

Artefactor releases are detailed in the
 [milestone page](https://github.com/appvia/artefactor/milestones).
