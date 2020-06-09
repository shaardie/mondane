# Mondane

## Build

The `Makefile` can help you build the binaries.
You need to have a proper golang enviroment.
Then you can run `make` to test and build the binaries.

## Run

The binaries are configured with environment variables.
There is a example env file in `env`.
If you want to run your binary with this env, this little command can help you, e.g. run the api server:

```
$ env $(grep -v '^\s*$\|^\s*\#' env | xargs) ./api-server
```
