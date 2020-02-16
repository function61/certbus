#!/bin/bash -eu

if [ ! -L "/usr/local/bin/certbus" ]; then
	ln -s /workspace/rel/certbus_linux-amd64 /usr/local/bin/certbus
fi

source /build-common.sh

BINARY_NAME="certbus"
COMPILE_IN_DIRECTORY="cmd/certbus"

standardBuildProcess

# buildstep packageLambdaFunction
