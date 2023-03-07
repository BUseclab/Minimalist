#!/bin/sh

echo "[*] Creating LIM-Minimalist (step-2) dockers"
cd ./step-2/
docker-compose rm -f
docker-compose build

echo "[*] Running LIM-Minimalist docker"
docker-compose up
