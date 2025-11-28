#  Internal Inventory tracker
    

Hi you reached the internal inventory tracker 

please feel free to g through all the trees and read the README file on how to operate the code 

# Latest Addition Testing 
   
   Unit Test was the latest "Late" Addition this is for extra points 

   in the console please type  make test/models for all the unit test for all the major additions for the go code 


# QuickStart Guide 

Hello and welcome to the Internal inventory Tracker Let me guide you through how to set up the Go side of the api 

This code based was designed and made for docker Docker version 27.5.1 or later with use of docker-compose version 1.29.2

Firstly if anyone else uses docker for their final projeect please ensure you docker compose down or stop all other docker services first before setting up the container 

My code uses make Files so once the other code is down all you need to do is 

cd into /Internal-inventory-tracker$ and type make up 

this wil trigger the docker compose file to start making a container specifically for my project and will automatically setup all services including go, migrate, mailpit, postgres and apache 

this also auto calls the run/api that starts up the go code as well for more details on what you can do please consolt the make file 

after you wait the services should be up and running 

feel free to go into other Github trees for more Read files on the json testing but the most important one needed is 

curl -X POST http://localhost:8081/api/v1/login -H "Content-Type: application/json" -d '{"email":"admin@example.com","password":"admin123"}'

TOKEN=$"Enter generated token here "

echo "Testing token: $TOKEN"


The token is the most important part of the crud process because each user will generate their own unique token inluding admins

the migrate service will create a default admin account already with the provided credencials above 

and also please ensure that the migrations follow in order from 001 to 004 