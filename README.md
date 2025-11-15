#  Email onbording and notifications

This wasnt a main feature I had in mind but since This is more of a closed off system I would like the Admins have more control over who is getting logged in and wo is getting onboared to the system 

# I DO NOT WANT RANDM PEOPLE JOINING THIS SYSTEM 

# Notifications
Since making the onboarding system i decided since that funconaity will be in place to also sends email notifications as well 

Admin and Operational managers- Will Get all notifications

IT will get asset manager and Ticket notifications

Staff wil ony get Ticket notifications 

agents and misc will get notification of there own ticket

#testing

# Set your admin token
TOKEN="your_admin_token_here"

echo "=== TESTING NOTIFICATION SYSTEM ==="

# 1. Create a ticket to trigger notifications
echo -e "\n1. Creating ticket to trigger notifications..."
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Notification - Keyboard Issue",
    "description": "Testing the notification system with a keyboard problem",
    "type": "it_help",
    "priority": "normal",
    "asset_id": 2,
    "is_internal": false
  }' \
  http://localhost:8081/api/v1/tickets

# 2. Check notifications for current user
echo -e "\n2. Checking notifications..."
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/notifications

# 3. Check unread count
echo -e "\n3. Checking unread count..."
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/notifications/unread-count

# 4. Get notification types
echo -e "\n4. Getting notification types..."
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/notifications/types

# 5. Create an asset to trigger asset notifications (admins/IT only)
echo -e "\n5. Creating asset to trigger admin/IT notifications..."
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "internal_id": "AM-K001",
    "asset_type": "Keyboard",
    "manufacturer": "Logitech", 
    "model": "K120",
    "model_number": "K120",
    "serial_number": "KB123456",
    "status": "IN_STORAGE"
  }' \
  http://localhost:8081/api/v1/assets

# 6. Check notifications again
echo -e "\n6. Checking notifications after asset creation..."
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/notifications

# 7. Mark all as read
echo -e "\n7. Marking all notifications as read..."
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/v1/notifications/read-all

# 8. Verify unread count is zero
echo -e "\n8. Verifying unread count is now zero..."
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/v1/notifications/unread-count

echo -e "\n✅ NOTIFICATION SYSTEM TEST COMPLETE"


# 3. Test with real email addresses
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Real Email Test",
    "description": "Testing real email delivery",
    "type": "it_help", 
    "priority": "high",
    "assigned_to": 5
  }' \
  http://localhost:8081/api/v1/tickets


# Rebuild your application
docker compose up --build

# Test 1: Admin creates user with custom password
echo "=== Test 1: Admin Creates User with Custom Password ==="
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "agent1",
    "full_name": "Agent One", 
    "email": "agent1@example.com",
    "password": "SecurePass123!",
    "role_id": 4,
    "send_email": true
  }' \
  http://localhost:8081/api/v1/users

# Test 2: Admin resets user password with custom password
echo -e "\n=== Test 2: Admin Resets Password with Custom Password ==="
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "new_password": "NewSecurePass456!",
    "send_email": true
  }' \
  http://localhost:8081/api/v1/users/5/reset-password

# Test 3: Admin resets password (auto-generate strong password)
echo -e "\n=== Test 3: Admin Resets Password (Auto-Generate) ==="
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "send_email": true
  }' \
  http://localhost:8081/api/v1/users/5/reset-password

# Test 4: Send credentials (auto-generates strong password)
echo -e "\n=== Test 4: Send Credentials ==="
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' \
  http://localhost:8081/api/v1/users/5/send-credentials

# Check Mailpit at http://localhost:8025 to see different email types!