#!/bin/bash

apt-get update && apt-get install libasound2-dev

chmod +x ./libimprobable_worker.so
chmod +x ./balancer
chmod +x ./server

./balancer -dev=false "$@"
