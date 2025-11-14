#  Asset Management System

This is one of the main features of this program 

Since Creating the User and the RBAC system Development Time of the following Features are going to be easier to develop

Features Left:

Asset Managment System - Full Features will be implemented 

Ticket System

Front End Development

Unit Test 

Metrics for both the Asset system and Ticket system 



# How to operate 

this is how to operate the asset managment system with Curl 

This will be for testing purposes before implementing a full stack website and app with flutter 

# Creating assets as an admin

# Set your admin token
TOKEN="your_admin_token_here"

# Test 1: Create PC asset with date
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "internal_id": "DPA-PC001",
    "asset_type": "PC",
    "manufacturer": "Dell",
    "model": "OptiPlex 7070",
    "model_number": "OP7070",
    "serial_number": "ABC123456",
    "status": "IN_STORAGE",
    "date_purchased": "2024-01-15"
  }' \
  http://localhost:8081/api/v1/assets

# Test 2: Create another PC
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "internal_id": "DPA-PC002", 
    "asset_type": "PC",
    "manufacturer": "HP",
    "model": "EliteDesk 800 G5",
    "model_number": "ED800G5",
    "serial_number": "XYZ789012",
    "status": "IN_USE",
    "in_use_by": 5,
    "date_purchased": "2024-02-20"
  }' \
  http://localhost:8081/api/v1/assets

# Test 3: Create a headset
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "internal_id": "AM-H001",
    "asset_type": "Headset", 
    "manufacturer": "Logitech",
    "model": "H390",
    "model_number": "H390",
    "serial_number": "HS123456",
    "status": "IN_STORAGE"
  }' \
  http://localhost:8081/api/v1/assets


# List all assets to see everything
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/assets

# Filter by type
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/v1/assets?type=PC"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/v1/assets?type=Monitor"

# Filter by status
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/v1/assets?status=IN_USE"
curl -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/v1/assets?status=IN_STORAGE"


# Get asset ID first (let's assume you have asset ID 1 - the PC)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/assets

# Add a maintenance service log for asset ID 1
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service_type": "MAINTENANCE",
    "performed_by": 1,
    "performed_at": "2024-11-13",
    "next_service_date": "2025-05-13",
    "notes": "Routine maintenance: cleaned fans, updated drivers, checked hardware"
  }' \
  http://localhost:8081/api/v1/assets/1/service-logs

# Add a repair service log
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "service_type": "REPAIR", 
    "performed_by": 5,
    "performed_at": "2024-10-15",
    "notes": "Replaced faulty RAM module"
  }' \
  http://localhost:8081/api/v1/assets/1/service-logs

# Get all service logs for asset ID 1
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/assets/1/service-logs

# Get specific service log (replace {id} with actual log ID from above)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/service-logs/1

# Check that asset service dates were updated
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/assets/1