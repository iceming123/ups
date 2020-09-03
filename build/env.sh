#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
upsdir="$workspace/src/github.com/truechain"
if [ ! -L "$upsdir/ups" ]; then
    mkdir -p "$upsdir"
    cd "$upsdir"
    ln -s ../../../../../. ups
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$upsdir/ups"
PWD="$upsdir/ups"

# Launch the arguments with the configured environment.
exec "$@"
