#!/bin/bash

minikube start
kubectl create secret generic aws-creds --from-literal=aws-id=$(cat ~/.aws/credentials | grep "id" | tail -n 1 | cut -d ' ' -f3) --from-literal=aws-secret=$(cat ~/.aws/credentials | grep "key" | tail -n 1 | cut -d ' ' -f3)
kubectl create secret generic regcred --from-file=.dockerconfigjson=$HOME/.docker/config.json  --type=kubernetes.io/dockerconfigjson
