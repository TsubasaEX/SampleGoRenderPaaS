#!/bin/bash

# Create or overwrite the .env file
cat << EOF > .env
db_postgres_host=10.8.9.50
db_postgres_port=5000
db_postgres_username=licensedbadm
db_postgres_password=1234567
db_postgres_dbname=licensedb
db_postgres_sslmode=disable

db_redis_host=10.8.9.50
db_redis_port=6379
db_redis_username=""
db_redis_password=123456
db_redis_ssl=false
EOF

echo ".env file generated successfully!"