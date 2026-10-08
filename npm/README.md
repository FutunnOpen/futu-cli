# @futunn/futu-cli

npm wrapper for the `futu` CLI.

```bash
npm install -g github:FutunnOpen/futu-cli
futu version
```

After the package is published to the npm registry:

```bash
npm install -g @futunn/futu-cli
```

Update a GitHub npm installation by reinstalling from GitHub:

```bash
npm install -g github:FutunnOpen/futu-cli
```

After npm registry publication:

```bash
npm update -g @futunn/futu-cli
```

The default GitHub Release source is `https://github.com/FutunnOpen/futu-cli`.
Set `FUTU_CLI_RELEASE_BASE` only when testing a fork or private release source.

To test a temporary prerelease:

```bash
FUTU_CLI_VERSION=v0.1.0-test.1 npm install -g .
```
