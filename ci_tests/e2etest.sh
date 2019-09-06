#!/usr/bin/env bash
function finish {
  # Your cleanup code here
  #ensure docke registry is stopped
  docker stop registry &>/dev/null || true
  #rm -rf $TMPDIR
}
trap finish EXIT

: "${ARTEFACTOR_IMAGE_VARS?}" "${ARTEFACTOR_DOCKER_REGISTRY?}" "${ARTEFACTOR_GIT_REPOS?}"
set -xe
TMPSRC=$(mktemp -d)
./bin/artefactor save --archive-dir "$TMPSRC" --logs

cd "$TMPSRC"
if [ ! -z "$E2E_BREAK_TESTS" ]; then
    #get all docker.tar files 
    files=(`ls *.docker.tar`)
    #truncate first file in list.
    cat /dev/null > "${files[0]}"
fi

TMPDST=$(mktemp -d)
./artefactor restore --dest-dir "$TMPDST" --logs 

cd "${TMPDST}/artefactor"
docker run -d --rm --name registry -p 5000:5000 registry:2
docker ps
env |grep -i registry
ARTEFACTOR_ARCHIVE_DIR="${TMPDST}/artefactor${TMPSRC}"
ARTEFACTOR_DOCKER_USERNAME=a ARTEFACTOR_DODCKER_PASSWORD=a ./"${TMPSRC}"/artefactor publish --archive-dir "${ARTEFACTOR_ARCHIVE_DIR}" --logs

docker stop registry
echo "env:" "${ARTEFACTOR_IMAGE_VARS}"
env |grep IMAGE
./"${TMPSRC}/artefactor" update-image-vars --archive-dir "${ARTEFACTOR_ARCHIVE_DIR}" --logs