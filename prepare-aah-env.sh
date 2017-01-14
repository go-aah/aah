#!/usr/bin/env bash

set -e
echo "Preparing aahframework custom environment for travis"

# export variables
export PATH=$GOPATH/bin:$PATH
export CURRENT_BRANCH="integration"
export AAH_SRC_PATH=$GOPATH/src/aahframework.org

# find out current git branch
if [[ "$TRAVIS_BRANCH" == "master" ]]; then
  CURRENT_BRANCH="master"
fi

# create aah src path
mkdir -p $AAH_SRC_PATH

# get aah dependencies modules
list_of_modules=$1
if [ -n "$list_of_modules" ]; then
  for module in $list_of_modules; do
    git_path="git://github.com/go-aah/$module"
    dest_path="$AAH_SRC_PATH/$module"
    echo "Fetching $git_path"
    git clone -b $CURRENT_BRANCH $git_path $dest_path
  done
fi

# go get dependencies
current_module=$2
current_module_path=$AAH_SRC_PATH/$current_module
cp -rf "$GOPATH/src/github.com/go-aah/$current_module" "$AAH_SRC_PATH"

# update travis build directory
export TRAVIS_BUILD_DIR=$current_module_path
echo "TRAVIS_BUILD_DIR: $TRAVIS_BUILD_DIR"

# cleanup and change directory
rm -rf $GOPATH/src/github.com/go-aah/*
cd $current_module_path
