#!/usr/bin/env bash
docker kill nameserver
docker rm nameserver
docker run --name nameserver --net dfs --ip 172.18.0.10 -t nameserver