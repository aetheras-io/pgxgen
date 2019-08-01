# pgxgen

Generate CRUD Struct binding Boilerplate using ```github.com/jackc/pgx```

## Usage

```sh
# recommended to use "gen" in the PackageName to signify code is robot generated
pgxgen init ${PackageName}

# modify ${PackageName}/config.toml with correct tables
(cd ${PackageName} && pgxgen generate)
```