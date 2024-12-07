# knowstore

A Kubernetes tool that calculates the disk usage of a pod and writes the result to its annotations

## Installation

See [example.yaml](example.yaml) for an example for in-cluster deployment.

## Usage

```shell
export NODE_NAME="example"

kubectl get pod -A --field-selector "spec.nodeName=$NODE_NAME,status.phase=Running" -o go-template --template '{{range .items}}{{.metadata.namespace}}/{{.metadata.name}}{{"\t"}}{{index .metadata.annotations "com.yankeguo.knowstore/ephemeral-storage.usage-pretty"}}{{"\n"}}{{end}}'
```

## Credits

GUO YANKE, MIT License
