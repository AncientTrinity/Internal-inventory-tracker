# Roles model

This repository is when i will put all the files needed for Roles.go it doesnt have authentication as of yet as it is just for testing


here are samples below 

# List roles
curl http://localhost:8081/api/v1/roles

# Create role
curl -X POST http://localhost:8081/api/v1/roles \
  -H "Content-Type: application/json" \
  -d '{"name":"Admin"}'

# Get role
curl http://localhost:8081/api/v1/roles/1

# Update role
curl -X PUT http://localhost:8081/api/v1/roles/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"IT"}'

# Delete role
curl -X DELETE http://localhost:8081/api/v1/roles/1
