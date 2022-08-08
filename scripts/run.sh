#!/bin/sh
echo building dynamic proxy...
go build

export $(cat .env)
FLAGS=$(
  echo "-add-http=$HTTP \
    -dir $DIR \
    -ia $INTERNAL_ADDRESS \
    -ip $INTERNAL_PORT \
    -icn $INTERNAL_CNAME \
    -ea $EXTERNAL_ADDRESS \
    -ep $EXTERNAL_PORT \
    -ecn $EXTERNAL_CNAME"
  )

tmux new-session -d "envoy -c bootstrap/local.yml"
tmux split-window -h "./dynamic-proxy $FLAGS"
tmux split-window -v -c "#{pane_current_path}/databags"
tmux -2 attach-session -d
