#!/bin/bash
# Starts the Java app (:8080) and the Go sidecar (:8090) in the same container.

java ${JAVA_OPTS} -jar /app/app.jar &
/app/sidecar &

# "docker stop" / Ctrl+C: pass the stop signal on to both processes.
trap 'kill -TERM $(jobs -p) 2>/dev/null' TERM INT

# Wait until either process exits (or a stop signal arrives), then stop the other one too.
wait -n
kill -TERM $(jobs -p) 2>/dev/null
wait
