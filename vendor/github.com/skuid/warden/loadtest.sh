#!/usr/bin/env bash

CONCURRENT=100
TIME=1m
SKUID_SESSIONID=$(get_sessionid) 

siege -l --concurrent=$CONCURRENT \
    -t$TIME \
    --content-type="application/json" \
    -H"x-skuid-session-id: $SKUID_SESSIONID" \
    "https://localhost:3004/api/v2/datasources/6f3eef71-6ac5-499d-ba4a-62e2866dacbf/load POST < $PWD/samples/load/post.json"

