# Design of Diag Collector & Checker for Kubernetes ([tidb-operator](https://github.com/pingcap/tidb-operator))
> For Diag Version 0.4.0

## Summary
Bundle current abilities of diag for `TiUP-Cluster` deployed clusters to the `diag-collector` Pod.  So that the user doesn't need to install [TiUP](https://github.com/pingcap/tiup) anymore.

## The Pod
Currently (`v0.3.0`), the `diag-collector` Pod is a one time job runner that receives arguments from environment variables and collects data for once. We’re going to make the Pod a long run container and the user does not need to destroy and recreate it every time they need to collect a new set of data.

- The Pod will be deployed into the same Namespace as `tidb-operator` (`tidb-admin` for example) instead of the Namespace of TiDB cluster
- The Pod will need higher RBAC permissions in order to retrieve information from outside of the Namespace of TiDB cluster
- The Pod will be an HTTP server that provides REST API for all operations, no more arguments in environment variables

## The API
Provide all diag abilities via REST APIs.

- Trigger a collect job with argument
- Query status of all jobs
- Manager collected datasets
- Trigger uploading of a dataset (package and upload)

## Workflow
The user creates a Deployment and a Service object in the same Namespace as `tidb-operator` (typically `tidb-admin`), as well as necessary RBAC roles. Then all operations could be done via the HTTP RESTful API.

## Usage
Users can access the Pod via HTTP RESTful API (via Service).

## Data Management
The user can manage collected data via HTTP RESTful API.

## API Design
Swagger specification file is at: https://github.com/pingcap/diag/blob/master/api/swagger.yaml

Use Swagger Editor to view it, or install `go-swagger` and use `swagger serve swagger.yaml`
