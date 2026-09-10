Awsdash is a CLI tool (and library) that creates or updates custom AWS CloudWatch dashboards.

```mermaid
flowchart LR
    subgraph AWS
        cli["$ awsdash"]
        resources["AmplifApps
ApiGateways
Lambdas
..."]
        dashboard["CloudWatch dashboard
- widget1
- widget2
..."]

        cli -->|"1) discover resources
[by tag]"| resources
        cli -->|"2) create/update"| dashboard
    end
```

Tool usage:

```sh
$ go install github.com/go-monk/awsdash@latest
$ awsdash -h
```
