#!/bin/sh

exec /usr/bin/fanctl -thermal-zone $(grep k10temp /sys/class/hwmon/hwmon*/name -l)
