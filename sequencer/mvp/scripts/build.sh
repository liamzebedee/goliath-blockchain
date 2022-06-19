#!/bin/sh
set -ex

echo Generating protobufs...

# docker run -v $PWD:/defs namely/protoc-all -f defs/*.proto -l go
protoc --go_out=. --go_opt=paths=source_relative sequencer/messages/defs.proto

echo Done.