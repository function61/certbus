#!/bin/bash -eu

if [ ! -L "/usr/local/bin/certbus" ]; then
	ln -s /workspace/rel/certbus_linux-amd64 /usr/local/bin/certbus
fi

source /build-common.sh

BINARY_NAME="certbus"
COMPILE_IN_DIRECTORY="cmd/certbus"

# TODO: one deployerspec is done, we can stop overriding this from base image
function packageLambdaFunction {
	if [ ! -z ${FASTBUILD+x} ]; then return; fi

	cd rel/
	cp "${BINARY_NAME}_linux-amd64" "${BINARY_NAME}"
	rm -f lambdafunc.zip
	zip lambdafunc.zip "${BINARY_NAME}"
	rm "${BINARY_NAME}"
}

standardBuildProcess

buildstep packageLambdaFunction
