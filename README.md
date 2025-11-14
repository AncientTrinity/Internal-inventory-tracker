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

# Create a ticket
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Ticket",
    "description": "Testing the fixed ticket system", 
    "type": "it_help",
    "priority": "normal",
    "is_internal": false
  }' \
  http://localhost:8081/api/v1/tickets

# Update ticket status
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "received",
    "completion": 10,
    "assigned_to": 5
  }' \
  http://localhost:8081/api/v1/tickets/1/status


  # Create a ticket with a linked asset

  echo -e "\n=== Creating Ticket Linked to PC Asset ==="
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "PC Performance Issue - Slow Boot Time",
    "description": "Agent reports PC takes over 5 minutes to boot up and applications load very slowly. Affecting productivity.",
    "type": "it_help",
    "priority": "high",
    "asset_id": 3,
    "is_internal": false
  }' \
  http://localhost:8081/api/v1/tickets


  # Adding A Comment 

  echo -e "\n=== Adding Comment ==="
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "comment": "Initial diagnosis: Running disk cleanup and checking for background processes.",
    "is_internal": false
  }' \
  http://localhost:8081/api/v1/tickets/3/comments


  # Updating Ticket Status 

  echo -e "\n=== Updating Ticket Status ==="
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "received",
    "completion": 15,
    "assigned_to": 5
  }' \
  http://localhost:8081/api/v1/tickets/3/status