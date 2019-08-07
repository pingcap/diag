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

cd analyzer
go build
/bin/cp analyzer $bin_dir
cd ..

cd api
go build
/bin/cp tidb-foresight $bin_dir
cd ..

cd spliter
go build
/bin/cp spliter $bin_dir
cd ..

cd syncer
go build
/bin/cp syncer $bin_dir
cd ..

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
