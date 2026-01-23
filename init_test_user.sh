#!/bin/bash

# Script to create test user in database
# This script creates an admin user for testing

echo "========================================="
echo "Initialize Test User"
echo "========================================="
echo ""

# Generate password hash using Go
echo "Generating password hash for 'admin123'..."
HASH=$(go run -C /home/ubuntu/go_cmdb_new <<'EOF'
package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	fmt.Print(string(hash))
}
EOF
)

if [ -z "$HASH" ]; then
    echo "Error: Failed to generate password hash"
    exit 1
fi

echo "Password hash generated successfully"
echo ""

# Insert user into database
echo "Inserting test user into database..."
mysql -h 20.2.140.226 -u root test <<EOF
-- Create users table if not exists
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) DEFAULT 'admin',
    status ENUM('active', 'inactive') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Delete existing admin user if exists
DELETE FROM users WHERE username = 'admin';

-- Insert test admin user
INSERT INTO users (username, password_hash, role, status)
VALUES ('admin', '$HASH', 'admin', 'active');
EOF

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Test user created successfully"
    echo ""
    echo "Credentials:"
    echo "  Username: admin"
    echo "  Password: admin123"
    echo "  Role: admin"
    echo "  Status: active"
    echo ""
else
    echo ""
    echo "✗ Failed to create test user"
    echo ""
    exit 1
fi

# Verify user creation
echo "Verifying user creation..."
mysql -h 20.2.140.226 -u root test -e "SELECT id, username, role, status, created_at FROM users WHERE username='admin';"

echo ""
echo "========================================="
echo "Initialization complete!"
echo "========================================="
