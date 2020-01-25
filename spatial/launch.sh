#!/bin/bash

apt-get update && apt-get install libasound2-dev

./server "$@"
