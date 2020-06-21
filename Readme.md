# Mondane

The `Makefile` can help you build the binaries.
You need to have a proper golang enviroment.

There is also a `docker-compose.yml` file to build and run the binaries in docker container.
The binaries are configured with environment variables.
There are env files for the different docker container and also a GnuPG-encryted `env.gpg` file, which has to be decrypted and sourced before the `docker-compose.yml can be run.
