FROM postgres:16

COPY init/ /docker-entrypoint-initdb.d/
COPY seed/init-customer.sql /docker-entrypoint-initdb.d/09-seed-customer.sql
COPY seed/init-vehicle.sql /docker-entrypoint-initdb.d/10-seed-vehicle.sql
