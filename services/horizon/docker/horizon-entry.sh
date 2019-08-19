#!/bin/sh

./horizon db init || echo "Horizon database initialization failed (possibly because it has been done before)"
./horizon