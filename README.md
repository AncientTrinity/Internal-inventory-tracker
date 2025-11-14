#  Ticketing System

This is one of the main features of this program 

Since Creating the User and the RBAC system Development Time of the following Features are going to be easier to develop

Features Left:

Asset Managment System - Full Features - implemented 

Ticket System -  working on features

Front End Development

Unit Test 

Metrics for both the Asset system and Ticket system 



# How to operate 

this is how to operate the Ticketing  with Curl 

This will be for testing purposes before implementing a full stack website and app with flutter 


# Set your token

login using the test admin to generate token

curl -X POST http://localhost:8081/api/v1/login -H "Content-Type: application/json" -d '{"email":"admin@example.com","password":"admin123"}'

TOKEN=$"Enter generated token here "

echo "Testing token: $TOKEN"