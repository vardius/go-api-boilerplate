resource "kubernetes_secret" "config" {
  metadata {
    name = "config-main"
    namespace = kubernetes_namespace.main.metadata.0.name
    labels = local.labels
    annotations = local.annotations
  }

  data = {
    APP_SECRET = "secret" #TODO: changeme
    APP_ENV = "production"
    APP_DOMAIN = "https://go-api-boilerplate.local"
    APP_SHUTDOWN_TIMEOUT = "5s"
    APP_EVENT_HANDLER_TIMEOUT = "120s"

    HTTP_SERVER_READ_TIMEOUT = "60s"
    HTTP_SERVER_WRITE_TIMEOUT = "60s"
    HTTP_SERVER_SHUTDOWN_TIMEOUT = "120s"

    HOST = "0.0.0.0"

    HTTP_PORT = 3000
    HTTP_ORIGINS = "https://go-api-boilerplate.local|https://api.go-api-boilerplate.local|http://localhost:3000|http://0.0.0.0:3000|http://127.0.0.1:3000"

    GRPC_PORT = 3001
    # if a client pings more than once every 5 minutes (default), terminate the connection
    GRPC_SERVER_MIN_TIME = "5m"
    # ping the client if it is idle for 2 hours (default) to ensure the connection is still active
    GRPC_SERVER_TIME = "2h"
    # wait 20 second (default) for the ping ack before assuming the connection is dea
    GRPC_SERVER_TIMEOUT = "20s"
    # send pings every 10 seconds if there is no activity
    GRPC_CLIENT_TIME = "10s"
    # wait 20 second for ping ack before considering the connection dea
    GRPC_CLIENT_TIMEOUT = "20s"

    COMMAND_BUS_BUFFER = "100"

    # wait 15 sec for oauth server to initialize
    OAUTH_INIT_TIMEOUT = "15s"

    MONGO_HOST = "mongodb"
    MONGO_PORT = 27017
    MONGO_USER = "root"

    # Please keep in mind that the custom username and database will only take effect
    # when the MongoDB pod runs for the first time:
    # https://github.com/bitnami/bitnami-docker-mongodb/blob/master/README.md#creating-a-user-and-database-on-first-run
    # If you already had a MongoDB PVC when configured them, these settings won't take effect
    mongodb-password = "password" #TODO: changeme
    mongodb-root-password = "password" #TODO: changeme
    mongodb-replica-set-key = "password" #TODO: changeme

    MAILER_HOST = "maildev"
    MAILER_PORT = 1025
    MAILER_USER = "go-api-boilerplate"
    MAILER_PASSWORD = "password"

    UI_BASE_URL = "https://api.go-api-boilerplate.local"
    AUTH_AUTHORIZE_URL = "https://go-api-boilerplate.local/authorize"

    AUTH_API_HOST = module.auth-api-main.name
    USER_API_HOST = module.user-api-main.name

    "tls.crt" = file("${path.module}/tls/server.pem")
    "tls.key" = file("${path.module}/tls/server.key")
  }

  type = "opaque"

  lifecycle {
    ignore_changes = [
      # ignore changes to labels and annotations as kubed adds/removes things there
      metadata,
    ]
  }
}

resource "kubernetes_secret" "regcred" {
  metadata {
    name = "docker-login"
    namespace = kubernetes_namespace.main.metadata.0.name
    labels = local.labels
    annotations = local.annotations
  }

  data = {
    ".dockerconfigjson" = file("${path.module}/.docker/config.json")
  }

  type = "kubernetes.io/dockerconfigjson"
}
