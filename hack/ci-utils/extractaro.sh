#!/bin/bash

set -xe

DOCKERID=$(docker create $1)
docker export $DOCKERID > aro.tar
tar -xvf aro.tar --strip-components=3 usr/local/bin/
docker rm $DOCKERID
