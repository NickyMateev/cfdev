#!/usr/bin/env bash
set -exo pipefail

dir="$( cd "$( dirname "$0" )" && pwd )"
home_dir=${CFDEV_HOME:-$HOME/.cfdev}
cache_dir="$home_dir/cache"
analyticskey="WFz4dVFXZUxN2Y6MzfUHJNWtlgXuOYV2"

export GOOS=darwin
export GOARCH=amd64

cfdevd="$PWD"/cfdvd
go build -o $cfdevd code.cloudfoundry.org/cfdev/cfdevd

analyticsd="$PWD"/analytix
analyticsdpkg="main"
go build \
  -o $analyticsd \
  -ldflags \
    "-X $analyticsdpkg.testAnalyticsKey=$analyticskey
     -X $analyticsdpkg.version=0.0.$(date +%Y%m%d-%H%M%S)" \
     code.cloudfoundry.org/cfdev/pkg/analyticsd

cfdepsUrl="$cache_dir/cfdev-deps.tgz"
pkg="code.cloudfoundry.org/cfdev/config"

go build \
  -ldflags \
    "-X $pkg.cfdepsUrl=file://$cfdepsUrl
     -X $pkg.cfdepsMd5=$(md5 $cfdepsUrl | awk '{ print $4 }')
     -X $pkg.cfdepsSize=$(wc -c < $cfdepsUrl | tr -d '[:space:]')

     -X $pkg.cfdevdUrl=file://$cfdevd
     -X $pkg.cfdevdMd5=$(md5 "$cfdevd" | awk '{ print $4 }')
     -X $pkg.cfdevdSize=$(wc -c < "$cfdevd" | tr -d '[:space:]')

     -X $pkg.analyticsdUrl=file://$analyticsd
     -X $pkg.analyticsdMd5=$(md5 "$analyticsd" | awk '{ print $4 }')
     -X $pkg.analyticsdSize=$(wc -c < "$analyticsd" | tr -d '[:space:]')

     -X $pkg.cliVersion=0.0.$(date +%Y%m%d-%H%M%S)
     -X $pkg.buildVersion=dev
     -X $pkg.testAnalyticsKey=$analyticskey" \
     code.cloudfoundry.org/cfdev


