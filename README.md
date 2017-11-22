# Service2Compose
Searching Docker stack ls and recreates docker compose from data

Really built for use in our local Docker environment, may be useful for others.

Usage:

Service2Compose -stack=<string>

will go through find all stacks matching string and attempt to rebuild compose file for that stack 

If not -stack specified then does all stacks it finds.

Assumes you are running inside a client bundle.
