# Service2Compose
Searching Docker stack ls and recreates docker compose from data

Really built for use in our local Docker environment, may be useful for others.

Usage:

Service2Compose -stack=<string> --unnamed --encrypt

will go through find all stacks matching string and attempt to rebuild compose file for that stack 

If not -stack specified then does all stacks it finds.
--unnamed Compose goes and creates a long stack name which it uses internally, if you use --unnamed you get what was given initially as the stack name
--encrypt Forces encryption of any of the networks that would be created with the compose

Assumes you are running inside a client bundle.
