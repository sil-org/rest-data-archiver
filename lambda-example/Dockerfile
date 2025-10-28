FROM node:22

ENV GO_VERSION=1.25.0

ADD https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip .
ADD https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz .

RUN <<EOF
  unzip awscli-exe-linux-x86_64.zip
  rm awscli-exe-linux-x86_64.zip
  ./aws/install
  rm -rf ./aws

  tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
  rm go${GO_VERSION}.linux-amd64.tar.gz
  ln -s /usr/local/go/bin/go /usr/local/bin/go

  npm install --ignore-scripts --global aws-cdk

  adduser user
EOF

USER user

WORKDIR /cdk
