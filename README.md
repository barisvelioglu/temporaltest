helm-charts % helm install --set server.replicaCount=1 --set cassandra.config.cluster_size=1 --set prometheus.enabled=false --set grafana.enabled=false --set elasticsearch.enabled=false temporaltest . --timeout 15m



https://learn.temporal.io/getting_started/go/first_program_in_go/