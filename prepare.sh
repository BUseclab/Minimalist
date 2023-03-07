#!/bin/sh
cur=$PWD
echo "[*] Download Minimalist packages (It takes a few seconds)"
wget -q https://www.dropbox.com/s/9mdhcu81gk55drw/packages.zip

echo "[*] Move Minimalist packages to step-1 directory"
mv packages.zip ./step-1/data/go-workspace/src/

echo "[*] Unzip Minimalist packages"
cd ./step-1/data/go-workspace/src/
unzip -o packages.zip
rm packages.zip

cd $cur
echo "[*] Download Less is More (LIM) docker files (It takes a few seconds)"
wget -q https://www.dropbox.com/s/79xbyzekiakabvd/step-2.zip

echo "[*] Unzip LIM"
unzip -o step-2.zip
rm step-2.zip

cd $PWD/step-1
echo "[*] Downloading phpMyAdmin 4.0.0"
wget -q  https://files.phpmyadmin.net/phpMyAdmin/4.0.0/phpMyAdmin-4.0.0-all-languages.zip

echo "[*] Unzipping the PMA"
rm -r ./data/4.0.0
unzip -q phpMyAdmin-4.0.0-all-languages.zip -d ./data
rm phpMyAdmin-4.0.0-all-languages.zip
mv ./data/phpMyAdmin-4.0.0-all-languages ./data/4.0.0

echo "[*] Copy config file"
cp ./data/config.inc.php ./data/4.0.0/config.inc.php

echo "[*] Set permissions"
chmod -R 777 ./data/4.0.0/
chmod 774 ./data/4.0.0/config.inc.php

cd $cur
echo "[*] copy phpMyAdmin to LIM-Minimalist directory"
cp -R $PWD/step-1/data/4.0.0/ $PWD/step-2/web/
chmod -R 777 $PWD/step-2/web/4.0.0
chmod 774 $PWD/step-2/web/4.0.0/config.inc.php
