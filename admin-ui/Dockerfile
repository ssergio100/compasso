FROM nginx:1.28-alpine@sha256:a8b39bd9cf0f83869a2162827a0caf6137ddf759d50a171451b335cecc87d236

RUN apk add --no-cache gettext \
    && rm -rf /usr/share/nginx/html/* \
    && install -d -o nginx -g nginx -m 0755 /opt/compasso

COPY --chown=nginx:nginx index.html api-client.js app.js style.css /usr/share/nginx/html/
COPY --chown=nginx:nginx runtime-config.template.js nginx.conf.template /opt/compasso/
COPY --chown=root:root docker-entrypoint.sh /usr/local/bin/compasso-admin-ui

RUN chmod 0555 /usr/local/bin/compasso-admin-ui \
    && chmod 0444 /usr/share/nginx/html/* /opt/compasso/*

USER nginx:nginx
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/compasso-admin-ui"]
