import requests
import json

BASE_URL = "http://localhost:8080/api/v1"

# Get tokens
login_admin = requests.post(f"{BASE_URL}/auth/login", json={"username": "admin", "password": "Admin@2024"})
login_user = requests.post(f"{BASE_URL}/auth/login", json={"username": "testuser001", "password": "Test123456"})

ADMIN_TOKEN = login_admin.json()['token']
USER_TOKEN = login_user.json()['token']

headers_admin = {"Authorization": f"Bearer {ADMIN_TOKEN}"}
headers_user = {"Authorization": f"Bearer {USER_TOKEN}"}

def test_api(name, method, url, headers=None, data=None):
    print(f"\n{'='*50}")
    print(f"Testing: {name}")
    print(f"Method: {method}")
    print(f"URL: {url}")
    if data:
        print(f"Data: {json.dumps(data, indent=2)}")
    
    try:
        if method == "GET":
            resp = requests.get(url, headers=headers)
        elif method == "POST":
            resp = requests.post(url, headers=headers, json=data)
        elif method == "PATCH":
            resp = requests.patch(url, headers=headers, json=data)
        
        print(f"Status: {resp.status_code}")
        try:
            print(f"Response: {json.dumps(resp.json(), indent=2)}")
        except:
            print(f"Response: {resp.text}")
    except Exception as e:
        print(f"Error: {e}")

print("="*50)
print("CHAIN2PLUS1 API TEST SUITE")
print("="*50)

# Public APIs
test_api("Public - Health Check", "GET", "http://localhost:8080/health")
test_api("Public - Product List", "GET", f"{BASE_URL}/admin/products")

# Auth APIs
test_api("Auth - Login", "POST", f"{BASE_URL}/auth/login", data={"username": "testuser001", "password": "Test123456"})

# User APIs (authenticated)
test_api("User - Profile", "GET", f"{BASE_URL}/user/profile", headers_user)
test_api("User - Tree", "GET", f"{BASE_URL}/user/tree", headers_user)

# Order APIs
test_api("Order - Create", "POST", f"{BASE_URL}/order/create", headers_user, {"product_id": 1, "payment_method": "test"})
test_api("Order - List", "GET", f"{BASE_URL}/order/list", headers_user)

# Profit APIs
test_api("Profit - List", "GET", f"{BASE_URL}/profit/list", headers_user)

# Withdraw APIs
test_api("Withdraw - Apply", "POST", f"{BASE_URL}/withdraw/apply", headers_user, {
    "amount": 10000,  # 100元 = 10000分
    "bank_name": "ICBC",
    "account_name": "testuser001",
    "account_no": "1234567890"
})
test_api("Withdraw - List", "GET", f"{BASE_URL}/withdraw/list", headers_user)

# Leaderboard APIs
test_api("Leaderboard - Total Earned", "GET", f"{BASE_URL}/leaderboard/total_earned", headers_user)
test_api("Leaderboard - Team Size", "GET", f"{BASE_URL}/leaderboard/team_size", headers_user)

# Admin APIs
test_api("Admin - Stats", "GET", f"{BASE_URL}/admin/stats", headers_admin)
test_api("Admin - Users", "GET", f"{BASE_URL}/admin/users", headers_admin)
test_api("Admin - Products", "GET", f"{BASE_URL}/admin/products", headers_admin)
test_api("Admin - Suppliers", "GET", f"{BASE_URL}/admin/suppliers", headers_admin)
test_api("Admin - Orders", "GET", f"{BASE_URL}/admin/orders", headers_admin)
test_api("Admin - Withdraw List", "GET", f"{BASE_URL}/admin/withdraw", headers_admin)

# Test recharge API
test_api("Recharge - User", "POST", f"{BASE_URL}/auth/recharge", headers_user, {
    "amount": 10000,
    "payment_mode": "wechat"
})

print("\n" + "="*50)
print("TEST COMPLETE")
print("="*50)