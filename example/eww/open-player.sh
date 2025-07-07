#!/bin/bash

# Usage: focus_mpris_by_pid.sh org.mpris.MediaPlayer2.spotify

if [ $# -eq 0 ]; then
    echo "Usage: $0 <player>"
    echo "Example: $0 spotify"
    exit 1
fi

player="$1"

# Get the PID of the MPRIS player from DBus
pid=$(dbus-send --session --print-reply --dest=org.freedesktop.DBus \
    /org/freedesktop/DBus \
    org.freedesktop.DBus.GetConnectionUnixProcessID \
    "string:org.mpris.MediaPlayer2.$player" |
    tail -1 |
    awk '{print $NF}')

if [ -z "$pid" ]; then
    echo "Could not get PID for player: $player"
    exit 1
fi

echo "Player PID:$pid"

# Get list of Hyprland clients and parse JSON
clients=$(hyprctl clients -j)

# Use jq to find the matching client by PID
workspace_id=$(echo "$clients" | jq -r ".[] | select(.pid == $pid) | .workspace.id")

if [ -z "$workspace_id" ] || [ "$workspace_id" = "null" ]; then
    echo "No window in Hyprland matches PID: $pid"
    exit 1
fi

echo "Player workspace:$workspace_id"

# Switch to the workspace
hyprctl dispatch workspace "$workspace_id"
