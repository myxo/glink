#!/bin/bash

set -e

echo Building go code...
go build cmd/*.go
echo Done

session="glink"

tmux new-session -d -s $session

# In my tmux settings, windows starts with '1', not '0'
window=1
user=alice
tmux rename-window -t $session:$window "main"
tmux send-keys -t $session:$window "./main -db-path=dbs/$user/$user.db" C-m

user=bob
tmux split-window -h "./main -db-path=dbs/$user/$user.db"


tmux attach-session -t $session
