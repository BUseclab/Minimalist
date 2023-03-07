#!/bin/sh

cd $PWD/auto_import/

echo "Copy step-1 results to auto-import direcctory"
cp ../step-1/result/allowed.txt .

echo "Create Python docker to run auto_import script"
docker build -q -t minimalist_auto_import .
rm allowed.txt

echo "Running the auto_import docker"
docker run --name minimalist-import --rm --network="host" -it minimalist_auto_import
docker rmi -f minimalist_auto_import


