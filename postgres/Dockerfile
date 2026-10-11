FROM postgres:16

# Runs once, on an empty data volume: creates the databases, plus schemas for
# services that don't have code (or migrations) yet. Built services apply
# their own migrations on startup.
COPY init/ /docker-entrypoint-initdb.d/
# Dev data, loaded after the services have migrated: sh /seed/seed.sh
COPY seed/ /seed/
