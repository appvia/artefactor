# Artefactor

[![Build_Status](https://circleci.com/gh/appvia/artefactor.svg?style=svg)](https://circleci.com/gh/appvia/artefactor)

Artefactor's primary use case is to enable "moving" a git repository and the
 artefacts it depends on when using a 
 [Sneakernet](https://en.wikipedia.org/wiki/Sneakernet).

- [Details](#details)
- [Usage](#usage)
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
| git-repos | [.] [local path] | Will archive a git repository. If the directory is the same as ${PWD}, it signifies the "home" for restoring. | `.` |
| docker-images | docker-image docker-image | A whitespace delimeted set of docker images | `mysql alpine` |
| web-files | url,filename,sha256[,true/false] | A whitespace seperated list of CSV's in the following format: </br></br>`url` is where to download from</br></br> `filename` is the name to save locally</br></br> `sha256` is the expected checksum</br></br>The optional last parameter specifies if the file should have executable permissions | `https://bit.ly/2ySXztI,kd,2f7...,true https://bit.ly/abc.iso,my.iso,abc...` |


*Common Flags:*
```
artefactor save --git-repos=.\
                --docker-images="mysql alpine" \
                --web-files https://github.com/UKHomeOffice/kd/releases/download/v0.13.0/kd_linux_amd64,kd,2f729bb26e225bcf61aa62a03d210f9a238d1c7b1666c1d72964decf7120466a,true
```
*Environment:*

```
export ARTEFACTOR_DOCKER_IMAGES="mysql
                                 alpine:latest"
export ARTEFACTOR_WEB_FILES="https://github.com/UKHomeOffice/kd/releases/download/v0.13.0/kd_linux_amd64,kd,2f729bb26e225bcf61aa62a03d210f9a238d1c7b1666c1d72964decf7120466a,true"

artefactor save
```

### restore

`artefactor restore` will restore artefacts to the original layout.
 (e.g. repo and ./downloads by default). It uses the meta data files stored with
 saved files to restore file permissions and structure.

*Common Flags:*

To restore a set of artefacter saved files from a home directory to the current
working directory:
```
artefactor restore --source-dir ~/
```

### publish

`artefactor publish` takes files from the relative ./downloads path and 
publishes containers / files to any remote registries / locations.

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
