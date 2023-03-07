#!/bin/sh


cd /home/result; /home/go-workspace/src/php-cg/scan-project/scan-project /home/pma-4.0.0/ database.db
echo "First analysis is done!"

DIR="result"

cd /home; ./extract_crawler_info.py -d -p /home/$DIR/ -I 172.19.0.1 -l /home/pma400_access.log -u /phpMyAdmin-4.0.0-all-languages/

echo "\n\n"
echo "The resulting files are in the result directory. Use the LIM docker to debloat the web app\n"
