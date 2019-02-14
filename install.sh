#!/usr/bin/env sh

VERSION=${1:-2.2.0}

shift

OUT=__canarycage
wget https://s3-us-west-2.amazonaws.com/loilo-public/oss/canarycage/${VERSION}/canarycage_linux_amd64.zip
mkdir -p ${OUT}
unzip canarycage_linux_amd64.zip -d ${OUT}
chmod +x ${OUT}/cage
mv ${OUT}/cage .
rm -rf ${OUT}
rm canarycage_linux_amd64.zip
echo "cage executable is installed at $(pwd)/cage"
