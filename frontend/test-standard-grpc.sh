#!/bin/bash
# This script demonstrates how to use the standard gRPC client in Node.js
# to connect to the Dropz Go backend service

# Change to the frontend directory
cd "$(dirname "$0")"

# Run a simple Node.js script to test the gRPC connection
node -e "
const { ServiceClient } = require('./src/proto/service_grpc_pb');
const { StatusRequest } = require('./src/proto/common_pb');
const grpc = require('@grpc/grpc-js');

// Create a client
const client = new ServiceClient('127.0.0.1:50051', 
                                grpc.credentials.createInsecure());

console.log('Connecting to gRPC server...');

// Create a request
const request = new StatusRequest();

// Make the call
client.getStatus(request, (error, response) => {
  if (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
  
  console.log('Connection successful!');
  console.log('Status:', response.toObject());
  process.exit(0);
});
"
