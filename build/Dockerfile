FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/buildly-service-discovery \
    USER_UID=1001 \
    USER_NAME=buildly-service-discovery

# install operator binary
COPY build/_output/bin/buildly-service-discovery ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
