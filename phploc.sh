#!/bin/sh

cd $PWD/phploc/

echo "[*] Create phploc docker to count line of code (It takes a few minutes)"
docker build -q -t minimalist_phploc .

echo "[*] Running the phploc docker"
cd ..
docker run --rm -v$PWD/step-2/web/4.0.0/:/home/4.0.0/ -it minimalist_phploc | grep "Logical Lines of Code"

echo "[*] Clean up"
docker rmi -f minimalist_phploc

