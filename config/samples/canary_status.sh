#!/bin/bash
while true;
  do
   kubectl -n canary-sample get canaries.cd.org.smart my-nginx-app -o=jsonpath="{.status}" | jq
   sleep 0.5
  done