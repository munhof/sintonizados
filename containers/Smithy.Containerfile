FROM docker.io/library/eclipse-temurin:21-jre
ARG SMITHY_VERSION=1.73.0
RUN apt-get update && apt-get install -y --no-install-recommends curl unzip ca-certificates && rm -rf /var/lib/apt/lists/*
RUN set -eu; case "$(uname -m)" in x86_64) arch=x86_64;; aarch64) arch=aarch64;; *) exit 1;; esac; \
    name="smithy-cli-linux-${arch}.zip"; \
    curl -fsSL "https://github.com/smithy-lang/smithy/releases/download/${SMITHY_VERSION}/${name}" -o "/tmp/${name}"; \
    curl -fsSL "https://github.com/smithy-lang/smithy/releases/download/${SMITHY_VERSION}/${name}.sha256" -o "/tmp/${name}.sha256"; \
    cd /tmp; sha256sum -c "${name}.sha256"; unzip -q "$name" -d /opt; mv "/opt/smithy-cli-linux-${arch}" /opt/smithy; rm "$name" "${name}.sha256"
ENV PATH="/opt/smithy/bin:${PATH}"
WORKDIR /workspace/api
ENTRYPOINT ["smithy"]
