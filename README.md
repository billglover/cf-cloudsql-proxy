# cf-cloudsql-proxy

There are a couple of options for deploying the [Google Cloud SQL Proxy](https://cloud.google.com/sql/docs/mysql/sql-proxy) on the Pivotal Platform  (PAS). This repository contains two sample deployments of the Cloud SQL Proxy.

The documentation for these deployments assumes that you have a Cloud SQL Instance up and running on GCP and that you are comfortable creating a service account with the appropriate permissions and generating the associated credentials file for the proxy to use. Deciding on the appropriate level of access for your Cloud SQL Proxy credentials or secure mechanisms for presenting credentials to your applications are not covered.

**Use Case:** You have an application running on the Pivotal Platform and you'd like to connect to a Cloud SQL instance via a Cloud SQL Proxy.

The application we are going to use to demonstrate the deployment of the Cloud SQL Proxy is a simple application written in Go that listens for incoming HTTP requests and responds by listing the MySQL databases it has access to. You will need the Go toolchain installed to be able to compile the sample application.

## Deployment

### Option 1: Sidecar Process
![Diagram](https://www.lucidchart.com/publicSegments/view/275431de-f3fb-445a-bacb-07d7a6e7bcd9/image.jpeg)

**Note:** Sidecar processes are available in Beta from PAS v2.6. Deployment of a sidecar process requires the use of the `cf v3-*` commands. These are subject to change and the documentation below may not represent the latest version of the commands.

Switch to the `sidecar` folder.

```plain
cd sidecar
```

Download the Linux version of the Cloud SQL Proxy. You need the Linux binary because it will ultimately run on the platform and not your local machine.

```plain
wget https://dl.google.com/cloudsql/cloud_sql_proxy.linux.amd64 -O cloud_sql_proxy
```

Mark the binary as executable.

```plain
chmod +x cloud_sql_proxy
```

Create a v3 application.

```plain
cf v3-create-app cloudsql-demo-sidecar --app-type buildpack
```

Compile a Linux version of the sample application.

```plain
GOOS=linux GOARCH=amd64 go build -o cloud_sql_demo
```

Replace the `key-sample.json` with the credentials for your Cloud SQL Proxy user. If you are unsure how to do this, see [Creating service account keys](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#iam-service-account-keys-create-gcloud). Ensure the file is named `key.json` when you are done.

```json
{
  "type": "service_account",
  "project_id": "[[REDACTED]]",
  "private_key_id": "[[REDACTED]]",
  "private_key": "-----BEGIN PRIVATE KEY-----[[REDACTED]]-----END PRIVATE KEY-----\n",
  "client_email": "[[REDACTED]]@[[REDACTED]].iam.gserviceaccount.com",
  "client_id": "[[REDACTED]]",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/[[REDACTED]]%40[[REDACTED]].iam.gserviceaccount.com"
}
```

Confirm your application folder mirrors the one shown below.

```plain
tree
.
├── cloud_sql_proxy
├── cloudsql-demo
├── key.json
├── main.go
└── manifest.yml

0 directories, 5 files
```

Modify the `manifest.yml` based on your Cloud SQL instance properties. Pay attention to the environment variables:

- `CLOUDSQL_USER` - the MySQL database username
- `CLOUDSQL_PASS` - the MySQL database password
- `CLOUDSQL_SOCKET_DIR` - the directory in which to create unix sockets
- `CLOUDSQL_INSTANCE` - the full Cloud SQL instance name as shown in the console
- `CLOUDSQL_PROJECT` the GCP project containing the Cloud SQL instance

Apply the application manifest.

```plain
cf v3-apply-manifest -f manifest.yml
```

Push the application.

```plain
cf v3-push cloudsql-demo-sidecar
```

Your application should now be available and return a list of databases available to the proxy.

```plain
~/c/j/c/u/app ❯❯❯ http https://cloudsql-demo-sidecar.apps.pcfone.io
HTTP/1.1 200 OK
Content-Length: 106
Content-Type: text/plain; charset=utf-8
Date: Wed, 14 Aug 2019 14:07:58 GMT
X-Vcap-Request-Id: 4bad5e7d-e5dc-42c1-6c23-9e65dbb4b746

Available databases:
--------------------
information_schema
mysql
performance_schema
sample_database
sys
```

### Option 2: User Provided Service

Switch to the `service/proxy` folder.

```plain
cd service/proxy
```

Download the Linux version of the Cloud SQL Proxy. You need the Linux binary because it will ultimately run on the platform and not your local machine.

```plain
wget https://dl.google.com/cloudsql/cloud_sql_proxy.linux.amd64 -O cloud_sql_proxy
```

Mark the binary as executable.

```plain
chmod +x cloud_sql_proxy
```

Replace the `key-sample.json` with the credentials for your Cloud SQL Proxy user. If you are unsure how to do this, see [Creating service account keys](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#iam-service-account-keys-create-gcloud). Ensure the file is named `key.json` when you are done.

```json
{
  "type": "service_account",
  "project_id": "[[REDACTED]]",
  "private_key_id": "[[REDACTED]]",
  "private_key": "-----BEGIN PRIVATE KEY-----[[REDACTED]]-----END PRIVATE KEY-----\n",
  "client_email": "[[REDACTED]]@[[REDACTED]].iam.gserviceaccount.com",
  "client_id": "[[REDACTED]]",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/[[REDACTED]]%40[[REDACTED]].iam.gserviceaccount.com"
}
```

Confirm your application folder mirrors the one shown below.

```plain
tree
.
├── cloud_sql_proxy
├── key.json
└── manifest.yml

0 directories, 3 files
```

Modify the `manifest.yml` based on your Cloud SQL instance properties. Pay attention to the environment variables:

- `CLOUDSQL_INSTANCE` - the full Cloud SQL instance name as shown in the consol
- `routes` - ensure you modify the domain here to use a TCP domain for your platform

Deploy the Cloud SQL Proxy.

```plain
cf push -f manifest.yml --random-port
```

Note the port that gets allocated to the application.

You now need to create a User Provided Service. Applications bound to this service will have access to the Cloud SQL Proxy along with database credentials (if included). You should probably opt to present the database credential to the application using a secrets vault or credential store.

```plain
cf create-user-provided-service cloudsql-proxy -p '{"db_host":"tcp.apps.pcfone.io","db_port":"10005", "db_user":"[[REDACTED]]", "db_pass":"[[REDACTED]]", "db_instance":"[[REDACTED]]"}'
```

You now need to build and deploy the sample application.

```plain
cd ../app
```

Compile a Linux version of the sample application.

```plain
GOOS=linux GOARCH=amd64 go build -o cloud_sql_demo
```

Push the application:

```
cf push -f manifest.yml
```

Your application should now be available and return a list of databases available to the proxy.

```plain
~/c/j/c/u/app ❯❯❯ http https://cloudsql-demo-app.apps.pcfone.io
HTTP/1.1 200 OK
Content-Length: 106
Content-Type: text/plain; charset=utf-8
Date: Wed, 14 Aug 2019 14:07:58 GMT
X-Vcap-Request-Id: 4bad5e7d-e5dc-42c1-6c23-9e65dbb4b746

Available databases:
--------------------
information_schema
mysql
performance_schema
sample_database
sys
```
