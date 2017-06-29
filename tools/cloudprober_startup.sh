#!/bin/bash -eu
#
# Copyright 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This startup_script makes it easy to run cloudprober inside a docker
# image.
while getopts 'i:j:v:' flag; do
  case "$flag" in
    i) IMAGE=${OPTARG} ;;
    j) JOB=${OPTARG} ;;
    v) VERSION=${OPTARG} ;;
  esac
done

shift $(($OPTIND - 1))
cmd=$1
if [ "$cmd" != "start" ]; then
  # First execution by cloud-init. Make the script accessible to all.
  chmod a+rx "$0"
  exit
fi

if [ -z "${JOB}" ]; then
  echo "-j <job_name> is a required parameter"
  exit 1
fi

# If IMAGE name is not defined, get it from the project name
if [ -z "${IMAGE}" ]; then
  PROJECT=$(curl -s http://metadata/computeMetadata/v1/project/project-id \
            -H "Metadata-Flavor: Google")
  IMAGE=gcr.io/${PROJECT}/${JOB}
fi

VERSION=${VERSION:-latest}
KERNEL_VERSION=$(uname -r)
GOOGLE_RELEASE=$(grep GOOGLE_RELEASE /etc/lsb-release|cut -d"=" -f2)

# Make sure that when this script exits, all child processes are exited as well
# Sending a SIGHUP/SIGTERM to bash kills the shell but leaves the child processes
# running. We don't want that behavior.
#   "kill -- -$$" sends a SIGTERM to the process group.
#   "trap - SIGTERM" removes the trap for SIGTERM to avoid recursion.
trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT
# Get credentials to fetch docker image from gcr.io
. /usr/share/google/dockercfg_update.sh
docker pull "${IMAGE}:${VERSION}"
DIGEST=$(docker inspect --format "{{.Id}}" "${IMAGE}:${VERSION}")
VARS="kernel=${KERNEL_VERSION},google_release=${GOOGLE_RELEASE}"
VARS="${VARS},${JOB}_tag=${VERSION},${JOB}_version=${DIGEST:0:12}"
docker run -e "SYSVARS=${VARS}" --net host \
  --privileged -v /tmp:/tmp "${IMAGE}:${VERSION}"
