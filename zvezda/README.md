# zvezda

Zvezda is the temporary solution for Airlock.

## Software

The software for zvezda consists of a simple go application, serving a website on port 3000. 
All endpoints of the webserver are secured using basic auth (for now).
Credentials are stored in a sqlite database in the working directory.
The website offers two buttons, which toggle the configured pin for the doors for 5 seconds, followed by a 5 second cooldown.

To build, a relative new version of go is required.

To run the webserver, use the `serve` command with the following environment variables:

```
PIN_INNERDOOR=16 PIN_OUTERDOOR=17 ./zvezda serve
```

To add users to zvezda, use:

```
zvezda user add <username>
```

The user will be asked for a password, which will be stored as a hash.
