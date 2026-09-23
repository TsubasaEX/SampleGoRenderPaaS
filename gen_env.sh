#!/bin/bash

# Create or overwrite the .env file
cat << EOF > .env
db_postgres_host=10.8.9.50
db_postgres_port=5000
db_postgres_username=licensedbadm
db_postgres_password=1234567
db_postgres_dbname=licensedb
EOF

echo ".env file generated successfully!"