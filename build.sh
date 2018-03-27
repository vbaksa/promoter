#!/bin/bash
#This file is used by Dockerfile
export GOPATH=/tmp
go build .
mkdir -p /opt/promoter
cp ./promoter /opt/promoter/
cp ./run.sh /opt/promoter/run.sh
chmod 755 /opt/promoter/promoter
chmod 755 /opt/promoter/run.sh

