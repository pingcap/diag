#!/bin/bash

joinArray() {
  local sep="\",\""
  local f="\"$1"
  shift
  printf "%s" "${f}" "${@/#/$sep}\""
}

while getopts c:n:m:d:s:f:t: flag
do
  case "${flag}" in
    c) cluster_name=${OPTARG};;
    n) cluster_namespace=${OPTARG};;
    m) monitor_namespace=${OPTARG};;
    d)
      _collectors=$(echo ${OPTARG} | tr "," "\n")
      collectors=$(joinArray ${_collectors})
      ;;
    s)
      _sql=()
      IFS=''
      while read line; do
        _sql+=("${line}")
      done < ${OPTARG}
      #echo ${_sql[@]}
      sql=$(joinArray ${_sql[@]})
      ;;
    f) from=${OPTARG};;
    t) to=${OPTARG};;
  esac
done

if [ -z ${monitor_namespace} ]; then
  monitor_namespace=${cluster_namespace}
fi

if [ -z ${to} ]; then
  to=`date "+%s"`
fi

#if [ -z ${from} ]; then
#  from=`date -d '2 hours ago' "+%s"`
#fi

data="{
  \"clusterName\": \"${cluster_name}\",
  \"namespace\": \"${cluster_namespace}\",
  \"monitor_namespace\": \"${monitor_namespace}\",
  \"collectors\": [${collectors}],
  \"explain_sqls\": [${sql}],
  \"from\": \"${from}\",
  \"to\": \"${to}\"
}"

echo "${data}"
curl -X POST "http://localhost:4917/api/v1/collectors" -d "${data}"

