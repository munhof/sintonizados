FROM docker.io/library/python:3.11-slim-bookworm AS build
ARG LAYA_REVISION=970dc8c5f63d7b886a68409493f37d569424f933
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*
RUN python -m venv /opt/venv
ENV PATH=/opt/venv/bin:$PATH PIP_NO_CACHE_DIR=1 PIP_DISABLE_PIP_VERSION_CHECK=1
COPY containers/laya-requirements.lock /tmp/laya-requirements.lock
RUN pip install --requirement /tmp/laya-requirements.lock --extra-index-url https://download.pytorch.org/whl/cpu
RUN git init /src && cd /src && git remote add origin https://github.com/NandhaKishorM/laya.git && git fetch --depth 1 origin ${LAYA_REVISION} && git checkout --detach FETCH_HEAD && test "$(git rev-parse HEAD)" = "${LAYA_REVISION}"
RUN pip install --no-deps --no-build-isolation '/src[serve]' && pip check

FROM docker.io/library/python:3.11-slim-bookworm
LABEL org.opencontainers.image.source="https://github.com/NandhaKishorM/laya"
ENV PATH=/opt/venv/bin:$PATH PYTHONUNBUFFERED=1 PYTHONDONTWRITEBYTECODE=1 USE_TF=0 USE_TORCH=1 TOKENIZERS_PARALLELISM=false LAYA_DEVICE=cpu LAYA_MODELS=multilingual LAYA_PRELOAD=1 LAYA_THREADS=4 HF_HOME=/home/laya/.cache/huggingface
RUN groupadd --gid 10001 laya && useradd --uid 10001 --gid laya --create-home laya && mkdir -p /home/laya/.cache/huggingface && chown -R laya:laya /home/laya/.cache
COPY --from=build /opt/venv /opt/venv
COPY --from=build /src/LICENSE /usr/share/doc/laya/LICENSE
USER laya
EXPOSE 8000
ENTRYPOINT ["laya-serve"]
