#!/usr/bin/env bash

which swagger>/dev/null || { echo "Please install go-swagger" ; exit 1;}

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

rm -rf "${scriptdir}/types"

swagger generate model \
    -f "${scriptdir}/swagger.yaml" \
    -t "${scriptdir}" \
    -m types

"${scriptdir}"/swagger-yaml-to-html.py < "${scriptdir}/swagger.yaml" > "${scriptdir}/doc.html"

