# Light Horizon services deployment

Light Horizon is composed of a few micro services:
* index-batch    - contains map and reduce binaries to parallize tx-meta reads and index writes.
* index-single   - contains single binary that reads tx-meta and writes indexes.
* ledgerexporter - contains single binary that reads from captive core and writes tx-meta
* web            - contains single binary that runs web api which reads from tx-meta and index.

See [godoc](https://godoc.org/github.com/stellar/go/exp/lighthorizon) for details on each service.

## Buiding docker images of each service
Each service is packaged into a Docker image, use the helper script included here to build:
`./build.sh <service_name> <dockerhub_registry_name> <tag_name> <push_to_repo[true|false]>`

example to build just the mydockerhubname/lighthorizon-index-single:latest image to docker local images, no push to registry:
`./build.sh index-single mydockerhubname latest false`

example to build images for all the services and push them to mydockerhubname/lighthorizon-<servicename>:testversion:
`./build.sh all mydockerhubname testversion true`

## Deploy service images on kubernetes(k8s) 
* `k8s/ledgerexporter.yml` - creates a deployment with ledgerexporter image and supporting resources, such as configmap, secret, pvc for captive core on-disk storage. Review the settings to confirm they work in your environment before deployment.
* `k8s/lighthorizon_index.yml` - creates a deployment with index-single image and supporting resources, such as configmap, secret. Review the settings to confirm they work in your environment before deployment.
* `k8s/lighthorizon_web.yml` - creates a deployment with the web image and supporting resources, such as configmap, ingress rule. Review the settings to confirm they work in your environment before deployment. 
