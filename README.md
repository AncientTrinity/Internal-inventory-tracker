This repository is when i will put all the files needed for user.go it doesnt have authentication as of yet as it is just for testing


here are samples below 

# List all users
curl http://localhost:8081/api/v1/users

# Create new user
curl -X POST http://localhost:8081/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username":"victor","full_name":"Victor Tillett","email":"victor@example.com","password":"secret","role_id":1}'

# Get one user
curl http://localhost:8081/api/v1/users/1

# Update
curl -X PUT http://localhost:8081/api/v1/users/1 \
  -H "Content-Type: application/json" \
  -d '{"username":"victor","full_name":"Victor A. Tillett","email":"victor@example.com","role_id":2}'

# Delete
curl -X DELETE http://localhost:8081/api/v1/users/1
