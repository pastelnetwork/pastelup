#!/bin/bash

echo "installing dd-service..."
pastelup install dd-service -no-cache -force

echo "starting dd-service..."
pastelup start dd-service

echo "successfully installed and started dd-service"