#  RBAC 

# Set your token


login using the test admin  to generate token 

curl -X POST http://localhost:8081/api/v1/login   -H "Content-Type: application/json"   -d '{"email":"admin@example.com","password":"admin123"}'


TOKEN=$"Enter generated token here "

# Test getting all users
echo "=== Testing Users Endpoint ==="
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/users

# Test getting all roles
echo -e "\n=== Testing Roles Endpoint ==="
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/roles

# Test getting specific user
echo -e "\n=== Testing Specific User ==="
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/users/1

# Test getting specific role
echo -e "\n=== Testing Specific Role ==="
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/roles/1
