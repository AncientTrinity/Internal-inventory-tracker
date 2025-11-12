#  Auth model 
 
 This repo is mostly used as a backup and moving foward each user can have a regular auth token or a protected auth token for the admins.



# 1. Login (Get JWT)
curl -X POST http://localhost:8081/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com", "password": "password123"}'

# Expected Reponse 

{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6..."
}


# 2. Access a Protected Route

Once you have your token, you can test a protected endpoint.


curl -X GET http://localhost:8081/api/v1/users \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"


# Expected result:

If the token is valid, you’ll get data.
If not, you’ll get:

{
  "error": "unauthorized"
}


# Refresh token
 
 curl -X POST http://localhost:8081/api/v1/refresh \
  -H "Content-Type: application/json" \
  -d '{"token": "YOUR_OLD_TOKEN_HERE"}'

{
  "token": "NEW_JWT_TOKEN_HERE"
}


# Need to be imlemented 

later on when i am at the mailing part i would like all users From admin to viewer to be authenticated via a token that would be used to access the application
Admins - Full admin rights to Create view Update and Delete- Tickets, Assets, Users 
Owner 
Management 
Hr 
IT 

low level Admins can Create View and Update Tickets only 
Staff 
Team leads/ QA 


Viewer would have view access only their ticket only 
Agents