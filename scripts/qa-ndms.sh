#!/bin/sh
# QA: NDMS users import endpoint
export PATH=/opt/sbin:/opt/bin:$PATH
CK=/tmp/ck5.txt
curl -s -c "$CK" -X POST http://127.0.0.1:8099/api/login \
  -H 'Content-Type: application/json' \
  -d '{"user":"admin","pass":"1235467890"}' >/dev/null
echo "=== ndms users"
curl -s -b "$CK" http://127.0.0.1:8099/api/ndms/users
echo
echo "=== bad login"
curl -s -X POST http://127.0.0.1:8099/api/login -H 'Content-Type: application/json' \
  -d '{"user":"admin","pass":"wrong"}'
echo
echo "=== unauth api"
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8099/api/status
