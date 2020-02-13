#!/bin/bash -eu

source /build-common.sh

BINARY_NAME="certbus"
COMPILE_IN_DIRECTORY="cmd/certbus"

standardBuildProcess

# buildstep packageLambdaFunction
