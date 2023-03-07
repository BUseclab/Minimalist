#!/bin/sh


Green='\033[0;32m'        # Green
NC='\033[0m'              # No color
mkdir /home/result
cd /home/result; /home/go-workspace/src/php-cg/scan-project/scan-project /home/webapp/ database.db

FILE="/home/result/calls.txt"

if test -f "$FILE"; then
	echo "##########################"
	echo "${Green}Basic Test was successful.${NC}"
	echo "##########################"
fi
