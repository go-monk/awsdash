Awsdash is a CLI tool (and library) that creates or updates custom AWS CloudWatch dashboards.

Tool usage:

```sh
$ go install github.com/go-monk/awsdash@latest
$ awsdash -h
```

## Explain metric widgets

Awsdash can render selected CloudWatch metric widgets and use a local Kronk
vision model to explain them in the terminal. Explain mode is read-only: it does
not create or update a dashboard.

Provide `-explain` more than once or use a comma-separated list:

```sh
$ awsdash \
    -tags environment=production \
    -explain lambda.errors \
    -explain s3.size \
    -explain-since 24h \
    -explain-model Qwen3.6-35B-A3B-UD-Q4_K_M
```

Use `-explain all` to explain every metric widget. Run `awsdash -h` for the
complete list of stable widget IDs and locally installed, validated vision
models. Run `kronk catalog list --local` to browse downloadable models.

The model must support vision. Awsdash uses the Kronk Go SDK directly and
reuses one loaded model for all selected widgets. Kronk installs missing runtime
libraries and downloads the selected model on first use; subsequent runs use the
local cache. Longer time ranges such as `24h` or more are useful for S3 storage
metrics, which are published daily.

Kronk's detailed model and inference logs are hidden by default. Use
`-explain-verbose` when troubleshooting. Explain mode only discovers resource
families needed by the selected widgets; for example, explaining an Amplify
widget does not enumerate Lambda or S3 resources.