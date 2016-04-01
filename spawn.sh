#!/usr/bin/env bash

(
    sleep 4
    echo "Slept 4"
) &

(
    sleep 5
    echo "Slept 5"
) &

sleep 10
echo DONE