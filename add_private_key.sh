#!/bin/bash
# Adds a private key, for use from CI/CD environments

mkdir -p ~/.ssh
chmod 700 ~/.ssh
echo -e "Host *\n\tStrictHostKeyChecking no\n\n" > ~/.ssh/config
echo "${SSH_PRIVATE_KEY}" > ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa
