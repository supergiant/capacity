#!/bin/bash

DOCKER_BUILD_ARGS="--build-arg=GIT_VERSION=${TAG} --build-arg=GIT_COMMIT=$(git rev-parse HEAD)"

echo "$TRAVIS_REPO_SLUG":"$TAG"
# build the docker container
echo "Building Docker container"
docker build ${DOCKER_BUILD_ARGS} --tag "$TRAVIS_REPO_SLUG":"$TAG" .

# if the tag is a release, a latest tag is also built
echo "Building latest tag"
if [[ "$TRAVIS_TAG" =~ ^v[0-9]. ]]; then
  docker tag "$TRAVIS_REPO_SLUG":"$TAG" "$TRAVIS_REPO_SLUG":"latest"
fi

if [ $? -eq 0 ]; then
	echo "Complete"
else
	echo "Build Failed"
	exit 1
fi
