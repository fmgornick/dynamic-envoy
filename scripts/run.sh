#!/bin/sh
echo building dynamic proxy...
go build

export $(cat .env)
tmux new-session -d "envoy -c bootstrap/local.yml"
tmux split-window -h "./dynamic-proxy -add-http=$HTTP -dir $DIR -ia $INTERNAL_ADDRESS -ip $INTERNAL_PORT -icn $INTERNAL_CNAME -ea $EXTERNAL_ADDRESS -ep $EXTERNAL_PORT -ecn $EXTERNAL_CNAME"
tmux split-window -v -c "#{pane_current_path}/databags"
tmux -2 attach-session -d
