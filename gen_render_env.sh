#!/bin/bash

# Create or overwrite the .env file
cat << EOF > .env
db_postgres_host=dpg-dapnq0gu01pc73das270-a.oregon-postgres.render.com
db_postgres_port=5432
db_postgres_username=school_6suf_user
db_postgres_password=nlMAlTE8bdbzCFGCUusURRAtd65g3aVe
db_postgres_dbname=school_6suf
db_postgres_sslmode=require

db_redis_host=oregon-keyvalue.render.com
db_redis_port=6379
db_redis_username=red-daq8cb17lnhs73c2ehog
db_redis_password=WFvOBMC7PqGWFTzpXiPufKbDSF9KCQpt
EOF

echo ".env file generated successfully!"