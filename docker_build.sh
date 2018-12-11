#!/bin/bash

echo "$TRAVIS_REPO_SLUG":"$TAG"
# build the docker container
echo "Building Docker container"
docker build ${DOCKER_BUILD_ARGS} --tag ${DOCKER_IMAGE_VERSIONED} .
docker tag ${DOCKER_IMAGE_VERSIONED} ${DOCKER_IMAGE_LATEST}

if [ $? -eq 0 ]; then
	echo "Complete"
else
	echo "Build Failed"
	exit 1
fi
