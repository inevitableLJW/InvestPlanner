FROM python:3.13.5-slim-bookworm

ARG AKTOOLS_VERSION=0.0.91
ARG AKSHARE_VERSION=1.18.64

RUN pip install --no-cache-dir \
    "aktools==${AKTOOLS_VERSION}" \
    "akshare==${AKSHARE_VERSION}" \
    "gunicorn==23.0.0" \
    "uvicorn==0.35.0"

WORKDIR /usr/local/lib/python3.13/site-packages/aktools
EXPOSE 8080
CMD ["gunicorn", "--bind", "0.0.0.0:8080", "--workers", "2", "--worker-class", "uvicorn.workers.UvicornWorker", "main:app"]

