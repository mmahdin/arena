#!/bin/sh
set -eu

# Override API_UPSTREAM with the backend's private Service URL when necessary.
: "${API_UPSTREAM:=http://ai-therapist-backend-service.ramona.svc.cluster.local:80}"
: "${NGINX_RESOLVER:=$(awk '/^nameserver / { print $2; exit }' /etc/resolv.conf)}"
export API_UPSTREAM NGINX_RESOLVER

# Preserve Nginx variables such as $uri, $request_uri and $http_host.
envsubst '${API_UPSTREAM} ${NGINX_RESOLVER}' \
    < /etc/nginx/templates/default.conf.template \
    > /etc/nginx/http.d/default.conf

exec "$@"
