#!/usr/bin/env bash
function finish {
  # Your cleanup code here
  #ensure docke registry is stopped
  docker stop registry &>/dev/null || true
  rm -rf $TMPDIR
}
trap finish EXIT
: "${E2E_BREAK_TESTS:=0}"
: "${ARTEFACTOR_IMAGE_VARS?}" "${ARTEFACTOR_DOCKER_REGISTRY?}" "${ARTEFACTOR_GIT_REPOS?}"
set -xe

./bin/artefactor save --logs

cd downloads

if E2E_BREAK_TESTS; then
# uncomment to break e2e test with validate src files are checksumed.
#get all docker.tar files 
files=(`ls *.docker.tar`)
#truncate first file in list.
cat /dev/null > "${files[0]}"
fi

TMPDIR=$(mktemp -d)
./artefactor restore --dest-dir $TMPDIR --logs 

cd "${TMPDIR}/artefactor"
docker run -d --rm --name registry -p 5000:5000 registry:2
docker ps
env |grep -i registry

ARTEFACTOR_DOCKER_USERNAME=a ARTEFACTOR_DODCKER_PASSWORD=a ./downloads/artefactor publish --logs

docker stop registry
echo "env:" ${ARTEFACTOR_IMAGE_VARS}
env |grep IMAGE
./downloads/artefactor update-image-vars --logs