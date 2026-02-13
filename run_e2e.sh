#!/bin/bash
export basePort="8081"
export metricsPath='/metrics'
export baseUrl="localhost:${basePort}"

source .env
# Test if we have cloudflare api/key variables configured.

# Run cloudflare-exporter
nohup ./cloudflare_exporter --listen="${baseUrl}" >/tmp/cloudflare-exporter-test.out 2>&1 &
export pid=$!
sleep 5

# Get metrics
curl -s -o /tmp/cloudflare_exporter_test_output http://${baseUrl}${metricsPath}
sleep 3

# Run Tests
venom run tests/basic_tests.yml

# Cleanup
rm venom*.log
kill ${pid}
