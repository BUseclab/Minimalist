#!/bin/sh

echo "[*] Copy Minimalist static analysis to init directory"
mkdir $PWD/basic_test/data
cp -r $PWD/step-1/data/go-workspace $PWD/basic_test/data/
cd $PWD/basic_test/

echo "[*] Creating Init docker (It takes a few minutes)"
docker build -q -t minimalist_init .

echo "[*] Running Minimalist docker"
docker run minimalist_init

echo "[*] Clean-up"
rm -rf $PWD/data

docker rmi -f minimalist_init
