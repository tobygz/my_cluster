#!/bin/sh

if [ -f ./3rd.zip ]; then
    rm ./3rd.zip
fi

/usr/bin/zip -r 3rd.zip viphxin \
    xtaci \
    golang/glog

chmod 0777 ./3rd.zip
