#!/bin/bash

if [ -z $RELEASE ]
then
  echo "RELEASE is not set, return"

  if [ -z ${DOCKERHUB_TOKEN-} ]
  then
      echo "DOCKERHUB_TOKEN is not set"
  else
      echo "DOCKERHUB_TOKEN is set"
  fi

  exit 0
fi

tag_prefix="${RELEASE%-*}"

if [ "$tag_prefix" == "exporter-0.0.1" ]; then
  tag="latest"
else
  tag="$tag_prefix"
fi

echo "Copying device-metrics-exporter artifacts and pushing docker image with tag:$tag"

setup_dir () {
    ls -al /device-metrics-exporter/
    BUNDLE_DIR=/device-metrics-exporter/output/
    mkdir -p $BUNDLE_DIR
}

copy_artifacts () {
    if [[ "$tag" == "latest" ]]; then
      DEBIAN_VERSION="0.0.1"
    elif [[ "$tag" == nic-v* ]]; then
      DEBIAN_VERSION="${tag#nic-v}"
    elif [[ "$tag" == collab-* ]]; then
      DEBIAN_VERSION="${tag#collab-}"
    else
      # Remove leading 'v' if present (e.g., v1.2.3 -> 1.2.3)
      DEBIAN_VERSION="${tag#v}"
    fi
    # # copy docker image ubi9.4
    cp /device-metrics-exporter/docker/device-metrics-exporter-latest.tar.gz $BUNDLE_DIR/device-metrics-exporter-latest-$RELEASE.tar.gz
    # # copy docker image ubi9.4
    # cp /device-metrics-exporter/docker/device-metrics-exporter-sriov-latest.tar.gz $BUNDLE_DIR/device-metrics-exporter-sriov-latest-$RELEASE.tar.gz
    # # copy docker image ubi9.6
    # cp /device-metrics-exporter/docker/device-metrics-exporter-ainic-latest.tar.gz $BUNDLE_DIR/device-metrics-exporter-ainic-latest-$RELEASE.tar.gz
    # # copy docker image azure coreos 3
    # #cp /device-metrics-exporter/docker/device-metrics-exporter-latest-azure.tar.gz $BUNDLE_DIR/device-metrics-exporter-latest-azure-$RELEASE.tar.gz
    # # copy docker mock image
    # cp /device-metrics-exporter/docker/device-metrics-exporter-mock-latest.tar.gz $BUNDLE_DIR/device-metrics-exporter-mock-latest-$RELEASE.tar.gz
    # copy debian ubuntu packages
    cp /device-metrics-exporter/bin/amdgpu-exporter_22.04_amd64.deb  $BUNDLE_DIR/amdgpu-exporter_$RELEASE~22.04_amd64.deb
    cp /device-metrics-exporter/bin/amdgpu-exporter_24.04_amd64.deb  $BUNDLE_DIR/amdgpu-exporter_$RELEASE~24.04_amd64.deb
    cp /device-metrics-exporter/bin/amdgpu-exporter-sriov_22.04_amd64.deb  $BUNDLE_DIR/amdgpu-exporter-sriov_$RELEASE~22.04_amd64.deb
    cp /device-metrics-exporter/bin/amdgpu-exporter-sriov_24.04_amd64.deb  $BUNDLE_DIR/amdgpu-exporter-sriov_$RELEASE~24.04_amd64.deb
    cp /device-metrics-exporter/bin/amdgpuifoe-exporter_22.04_amd64.deb  $BUNDLE_DIR/amdgpuifoe-exporter_$RELEASE~22.04_amd64.deb
    cp /device-metrics-exporter/bin/amdgpuifoe-exporter_24.04_amd64.deb  $BUNDLE_DIR/amdgpuifoe-exporter_$RELEASE~24.04_amd64.deb 
    # # copy NIC debian ubuntu packages
    # cp /device-metrics-exporter/bin/amdnic-exporter_22.04_amd64.deb  $BUNDLE_DIR/amdnic-exporter_$RELEASE~22.04_amd64.deb
    # cp /device-metrics-exporter/bin/amdnic-exporter_24.04_amd64.deb  $BUNDLE_DIR/amdnic-exporter_$RELEASE~24.04_amd64.deb
    # copy rpm rhel9 packages
    cp /device-metrics-exporter/bin/amdgpu-exporter-rhel9.x86_64.rpm $BUNDLE_DIR/amdgpu-exporter-$RELEASE.x86_64.rpm
    cp /device-metrics-exporter/bin/amdgpu-exporter-sriov-rhel9.x86_64.rpm $BUNDLE_DIR/amdgpu-exporter-sriov-$RELEASE.x86_64.rpm
    cp /device-metrics-exporter/bin/amdgpuifoe-exporter-rhel9.x86_64.rpm $BUNDLE_DIR/amdgpuifoe-exporter-$RELEASE.x86_64.rpm
    # # copy helm charts
    # cp /device-metrics-exporter/build/charts/internal-device-metrics-exporter-charts.tgz $BUNDLE_DIR/internal-device-metrics-exporter-charts-$RELEASE.tgz
    # cp /device-metrics-exporter/build/charts/amdpsdo-device-metrics-exporter-charts.tgz $BUNDLE_DIR/amdpsdo-device-metrics-exporter-charts-$RELEASE.tgz
    # cp /device-metrics-exporter/build/charts/internal-nic-device-metrics-exporter-charts.tgz $BUNDLE_DIR/internal-nic-device-metrics-exporter-charts-$RELEASE.tgz
    # cp /device-metrics-exporter/build/charts/amdpsdo-nic-device-metrics-exporter-charts.tgz $BUNDLE_DIR/amdpsdo-nic-device-metrics-exporter-charts-$RELEASE.tgz
    # # copy techsupport scripts
    cp /device-metrics-exporter/tools/techsupport_dump.sh $BUNDLE_DIR/
    # list the artifacts copied out
    ls -la $BUNDLE_DIR
}

docker_push () {
    EXPORTER_IMAGE_URL=registry.test.pensando.io:5000/device-metrics-exporter
    EXPORTER_SRIOV_IMAGE_URL=registry.test.pensando.io:5000/device-metrics-exporter-sriov
    EXPORTER_MOCK_IMAGE_URL=registry.test.pensando.io:5000/device-metrics-exporter-mock
    EXPORTER_AINIC_IMAGE_URL=registry.test.pensando.io:5000/device-metrics-exporter-ainic

    # rhel 9 image push
    docker load -i /device-metrics-exporter/docker/device-metrics-exporter-latest.tar.gz
    docker inspect $EXPORTER_IMAGE_URL:latest | grep "HOURLY"
    docker tag $EXPORTER_IMAGE_URL:latest $EXPORTER_IMAGE_URL:$tag
    docker push $EXPORTER_IMAGE_URL:$tag

    # mock exporter image push
    docker load -i /device-metrics-exporter/docker/device-metrics-exporter-mock-latest.tar.gz
    docker inspect $EXPORTER_MOCK_IMAGE_URL:latest | grep "HOURLY"
    docker tag $EXPORTER_MOCK_IMAGE_URL:latest $EXPORTER_MOCK_IMAGE_URL:$tag
    docker push $EXPORTER_MOCK_IMAGE_URL:$tag

    # azurelinux3 image push
    #azuretag="$tag-azl3"
    #docker load -i /device-metrics-exporter/docker/device-metrics-exporter-latest-azure.tar.gz
    #docker inspect $EXPORTER_IMAGE_URL:latest | grep "HOURLY"
    #docker tag $EXPORTER_IMAGE_URL:latest $EXPORTER_IMAGE_URL:$azuretag
    #docker push $EXPORTER_IMAGE_URL:$azuretag

    # rhel 9 sriov image push
    docker load -i /device-metrics-exporter/docker/device-metrics-exporter-sriov-latest.tar.gz
    docker inspect $EXPORTER_SRIOV_IMAGE_URL:latest | grep "HOURLY"
    docker tag $EXPORTER_SRIOV_IMAGE_URL:latest $EXPORTER_SRIOV_IMAGE_URL:$tag
    docker push $EXPORTER_SRIOV_IMAGE_URL:$tag

    # rhel 9 ainic image push
    docker load -i /device-metrics-exporter/docker/device-metrics-exporter-ainic-latest.tar.gz
    docker inspect $EXPORTER_AINIC_IMAGE_URL:latest | grep "HOURLY"
    docker tag $EXPORTER_AINIC_IMAGE_URL:latest $EXPORTER_AINIC_IMAGE_URL:$tag
    docker push $EXPORTER_AINIC_IMAGE_URL:$tag

    if [ -z $DOCKERHUB_TOKEN ]
    then
      echo "DOCKERHUB_TOKEN is not set"
    else
      docker login --username=shreyajmeraamd --password-stdin <<< $DOCKERHUB_TOKEN
      # rhel 9
      docker tag $EXPORTER_IMAGE_URL:$tag amdpsdo/device-metrics-exporter:$RELEASE
      docker push amdpsdo/device-metrics-exporter:$RELEASE
      # sriov rhel 9
      docker tag $EXPORTER_SRIOV_IMAGE_URL:$tag amdpsdo/device-metrics-exporter-sriov:$RELEASE
      docker push amdpsdo/device-metrics-exporter-sriov:$RELEASE
      # ainic rhel 9
      docker tag $EXPORTER_AINIC_IMAGE_URL:$tag amdpsdo/device-metrics-exporter-ainic:$RELEASE
      docker push amdpsdo/device-metrics-exporter-ainic:$RELEASE
      # azure linux3
      #docker tag $EXPORTER_IMAGE_URL:$azuretag amdpsdo/device-metrics-exporter:$RELEASE-azl3
      #docker push amdpsdo/device-metrics-exporter:$RELEASE-azl3
    fi
}

setup () {
    setup_dir
    copy_artifacts
    #docker_push
}

upload () {
    cd $BUNDLE_DIR
    find . -type f -print0 | while IFS= read -r -d $'\0' file;
      do asset-push builds hourly-device-metrics-exporter $RELEASE "$file" ;
      if [ $? -ne 0 ]; then
        exit 1
      fi
    done
}

main () {
  setup
  upload
}

main
