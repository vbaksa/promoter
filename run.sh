#!/bin/bash

if [ "x${SRC_IMAGE}" == "x" ]
  then
    echo "Warning environment variable SRC_IMAGE is not set"
  fi
if [ "x${DEST_IMAGE}" == "x" ]
  then
    echo "Warning environment variable DEST_IMAGE is not set"
  fi
if [ "x${SRC_USERNAME}" == "x" ]
  then
    SRC_USERNAME="builder"
  fi
if [ "x${DEST_USERNAME}" == "x" ]
  then
    DEST_USERNAME="builder"
  fi
if [ "x${SRC_PASSWORD}" == "x" ]
  then
     if [ -f /var/run/secrets/kubernetes.io/serviceaccount/token ]; then
	echo "Using Pod Token authentication"
        SRC_PASSWORD=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
        fi
  fi
if [ "x${DEST_PASSWORD}" == "x" ]
  then
     if [ -f /var/run/secrets/kubernetes.io/serviceaccount/token ]; then
	echo "Using Pod Token authentication"
        DEST_PASSWORD=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
        fi
  fi
if [ "x${APP_ARGS}" == "x" ]
  then
    APP_ARGS="${SRC_IMAGE} ${DEST_IMAGE} --src-username=${SRC_USERNAME} --src-password=${SRC_PASSWORD} --dest-username=${DEST_USERNAME} --dest-password=${DEST_PASSWORD} ${ADDITIONAL_FLAGS}"

  fi
echo "Running ${APP_ARGS}"
/opt/promoter/promoter push ${APP_ARGS}

