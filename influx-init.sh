#!/bin/bash
set -e

INFLUX_URL="${INFLUX1_URL:-http://localhost:8086}"
INFLUX_TOKEN="${INFLUX1_TOKEN:-q2-influx-token}"
INFLUX_ORG="${INFLUX1_ORG:-q2org}"

influx bucket create --host "$INFLUX_URL" --token "$INFLUX_TOKEN" --org "$INFLUX_ORG" --name "realtime" --retention 7d 2>/dev/null || true
influx bucket create --host "$INFLUX_URL" --token "$INFLUX_TOKEN" --org "$INFLUX_ORG" --name "history" --retention 365d 2>/dev/null || true
echo "influx init done"
