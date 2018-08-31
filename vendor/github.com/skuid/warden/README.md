[![Build Status](https://overwatch.skuid.ink/api/badges/skuid/warden/status.svg)](https://overwatch.skuid.ink/skuid/warden)

# 💂 warden

This project is a small Go service to proxy data source requests and enforce Data Source Object regulations.

## API Usage, v2

V2 is an attempt at a more RESTful service. It also uses stores information about datasources and permissions rather than having to call Pliny to get that information. See [the Warden v2 architecture doc][v2architecture] for more information.

### Some routes:

* `/api/v2/datasource` - get, put, post, and delete a datasource object. Also, patch to pull the entity information from SeaQuill and create a datasource.
* `/api/v2/datasource/<ds_id>/entity` - get, put, post, delete, and patch an entity in a datasource.
* `POST /api/v2/datasource/<ds_id>/load` - Using Skuid's load language, load a set of entities (tables). Matches the `/api/v1/load` route, more or less.
* `POST /api/v2/datasource/<ds_id>/save` - Using Skuid's save language, runs a set of CRUD operations against entities (tables). Matches the `/api/v1/save` route, more or less.

### Headers

You must provide the `Content-type` and `Accepts` headers. Currently, only `application/json`

For authorization, Warden expects JWT tokens to be provided using the `Authorization: bearer <jwt_token>` header. V1 routes still support `x-skuid-session-id`, but this is deprecated and V1 routes will eventually be removed.

Each `organization` record in Warden needs to have a corresponding `site_jwt_key` record in Warden's database with a valid RSA Public Key in the `public_key` column. This is setup by logging in to a Skuid Platform Site / Salesforce Org, selecting a JWT Signing Certificate, creating a Data Service record for the Warden instance you want to connect to, and clicking "Register Site with Data Service". This will cause Warden to fetch the Site/Org's JWT Signing Certificate's public key, and create a `site_jwt_key` record in the database. After this you should be able to make requests to Warden using JWT's issued by the Skuid Platform Site / Salesforce Org that are signed using RS256 with the Site/Org's private key.

### Examples

_All examples assume you are currently in the warden directory and have configured a site_

Get a list of datasources:

```bash
curl -XGET \
	-H"content-type: application/json" \
	-H"Accept: application/json" \
	-H"Authorization: bearer <JWT Signed with Private Key>" \
	https://localhost:3004/api/v2/datasources
```

Get information about a datasource:

```bash
curl -XGET \
	-H"content-type: application/json" \
	-H"Accept: application/json" \
	-H"Authorization: bearer <JWT Signed with Private Key>" \
	https://localhost:3004/api/v2/datasources/4699CBF9-61AE-4AFE-AAD2-72851E0C32A4
```

Call load using one of the sample post bodies:

```bash
curl -XPOST \
	-d@samples/load/post.json \
	-H"content-type: application/json" \
	-H"Accept: application/json" \
	-H"Authorization: bearer <JWT Signed with Private Key>" \
	https://localhost:3004/api/v2/datasources/4699CBF9-61AE-4AFE-AAD2-72851E0C32A4/load
```

## API Usage, v1 (deprecated)

This service supports several endpoints for interacting with Data Sources

##### `/api/v1/save` - Submit transformations for one or more data models
##### `/api/v1/load` - Load data model records
##### `/api/v1/getModelMetadata` - Load data model schema / metadata
##### `/api/v1/getEntityList` - Load a flat list of data model names
##### `POST /api/v1/getSourceEntityMetadata` - Load schema / metadata for a model from the datasource
##### `GET /api/v1/getSourceEntityList` - Load a flat list of models that exist in the datasource

For all calls pass the following headers:

```
content-type: application/json
x-skuid-data-source: <datasource name>
x-skuid-session-id: <user's session id> // Or, Authorization: bearer. See notes on JWT under the v2 usage.
```

user's session id can be found in a logged in console by typing `skuid.utils.userInfo.sessionId`

### Some examples:

**Get a list of tables (entities)**

```bash
curl \
	-H"content-type: application/json" \
	-H"x-skuid-data-source: pg_dvdrental" \
	-H"x-skuid-session-id: e9623886-4608-4418-a64d-c58295f3198c" \
	https://localhost:3004/api/v1/getSourceEntityList
```

**Get metadata about an entity**

```bash
curl \
	-H"content-type: application/json" \
	-H"x-skuid-data-source: pg_dvdrental" \
	-H"x-skuid-session-id: e9623886-4608-4418-a64d-c58295f3198c" \
	-d'{"entity":"actor"}' \
	https://localhost:3004/api/v1/getSourceEntityMetadata
```

## Development
### Pre-Setup
You must have Go installed:

```
brew install go
mkdir -p ~/go/{src,bin,pkg}
export GOPATH=~/go
# Append GOPATH to profile
echo 'export GOPATH=~/go' | tee -a ~/.profile
```

### Installation

```bash
mkdir -p $GOPATH/src/github.com/skuid/warden
git clone git@github.com:skuid/warden.git $GOPATH/src/github.com/skuid/warden
cd $GOPATH/src/github.com/skuid/warden
```

### Migrations

If there were any changes to the database you'll need to run a migration. We have a `migrate` command for warden. Running locally you'll need the `--debug=true` flag. Typically you'll just need to run either `./warden migrate up --debug=true` or `./warden migrate down --debug=true`

```
make migrate
```

### Build

```bash
go build
```

### Run

If you want to use environment variables, you can add these to your environment:
```
export WARDEN_PLINY_ADDRESS=https://localhost:3000
export WARDEN_QUILL_ADDRESS=http://localhost:3113
export WARDEN_SEAQUILL_ADDRESS=http://localhost:3113
export WARDEN_TLS_ENABLED=true
export WARDEN_TLS_CERT_FILE=certs/pliny.pem
export WARDEN_TLS_KEY_FILE=certs/pliny-key.pem
export WARDEN_PORT=3004
export WARDEN_PPROF=true
export WARDEN_DATABASE_ON=true
export WARDEN_LOCAL_AWS_CONF_ENABLED=true

export PGHOST=127.0.0.1
export PGPORT=15433
export PGDATABASE=warden
export PGUSER=warden
export PGPASSWORD=wardenDBpass

export REDIS_HOST=localhost
export REDIS_PORT=16379

export WARDEN_USE_KMS=false
export WARDEN_ENCRYPTION_KEY=EOXsUvCZCSCkvVsA8hYZFvd0p82vqqea
```

**note**: `WARDEN_ENCRYPTION_KEY` needs to be 32 bytes long. To generate your own, you can run:
```
cat /dev/urandom | LC_ALL=C tr -c -d 'a-zA-Z0-9' | fold -w 32 | head -n 1
```
Otherwise just use the one listed for local development.

If you want to use Pliny with warden, you'll need to add this environment variable to pliny:

```bash
export PLINY_WARDEN_HOST_NAME=https://localhost:3004
```

To start warden running, assuming you have the above environment variables set:

```bash
./warden serve
```

If you prefer make:
```bash
make start
```

If you don't want to use environment variables or `make`, you can run warden with command line flags.
The number of options necessary to run warden has grown such that this is not the recommended method for passing options into warden. For example:

```bash
./warden serve -p 3004 --pprof=true --pliny_address=https://localhost:3000 --tls_enabled=true --tls_cert_file="certs/pliny.pem" --tls_key_file="certs/pliny-key.pem" --database_on=true --local_aws_conf_enabled=true
```


### Test

Run `make test` to run the tests. It's a good idea to do this prior to pushing to github.

### Profile with pprof

To use some of the graphical stuff with pprof, install the [graphviz](https://www.graphviz.org/) dependency. For mac users, you can just run `brew install graphviz`.

To profile the running application with [pprof](https://golang.org/pkg/runtime/pprof/), start warden server with the `--pprof=true` flag (which `make start` will do for you). You can then profile and trace the running application using the pprof routes.

There is a `make profile` that will start collecting a profile and then get dumped into the pprof repl once stats are gathered. To get a trace you could do:

```bash
curl -so trace.out "https://localhost:3004/debug/pprof/trace?seconds=15"
go tool trace trace.out
```

__note__ pprof works best when there is load. You can just run a few test commands using either siege, locust, or just put a curl command in a `watch`.

### Dependencies

Dependencies are managed with [dep](https://github.com/golang/dep).

[v2architecture]: https://docs.google.com/document/d/1ByA8xXXWYMS4ud-SCr2XCKwqio3Ix5N5tblkH6h9rfA 
[jwt.io]: https://jwt.io
