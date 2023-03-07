#!/bin/sh

cd $PWD/step-1

echo "[*] Creating Minimalist (step-1) docker (It takes a few minutes)"
docker build -q -t minimalist .

echo "[*] Setting up environment for running the docker"
rm -rf result
mkdir result

echo "[*] Running Minimalist docker"
docker run -v $PWD/result:/home/result minimalist

echo "[*] Clean-up"
rm -f $PWD/result/database.db $PWD/result/fanout_output.json $PWD/result/output.log $PWD/result/functions.txt $PWD/result/calls.txt $PWD/result/methods.txt $PWD/result/unresolved.txt
docker rmi -f minimalist
