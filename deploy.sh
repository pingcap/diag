#!/usr/bin/bash

deploy_dir="/home/tidb/tidb-foresight/"
bin_dir=$deploy_dir"bin/"
script_dir=$deploy_dir"script/"
web_dir=$deploy_dir"web/"

mkdir -p $deploy_dir
mkdir -p $bin_dir
mkdir -p $script_dir
mkdir -p $web_dir

systemctl stop foresight

go build -o=$bin_dir"tidb-foresight" ./api

go build -o=$bin_dir"analyzer" ./analyzer

go build -o=$bin_dir"spliter" ./cmd/spliter

go build -o=$bin_dir"syncer" ./cmd/syncer

/bin/cp -r ./collector $script_dir
ln -sf $script_dir"collector/collector" $bin_dir"collector"

/bin/cp ./pioneer/pioneer.py $bin_dir"pioneer"

cd web
yarn
yarn build
rm -rf $web_dir"*"
/bin/cp -r dist/* $web_dir

cd $deploy_dir
chmod -R 777 *
chown -R tidb:tidb *

systemctl start foresight
