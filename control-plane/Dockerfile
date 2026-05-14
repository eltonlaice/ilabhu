# syntax=docker/dockerfile:1
#
# ilabhud — control-plane image.
#
# Multi-stage so the published image carries no Go toolchain, only the
# ilabhud binary plus the runtime tools it shells out to (terraform,
# kubectl, ssh, ssh-keygen, scp).

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /out/ilabhud ./cmd/ilabhud

FROM debian:bookworm-slim AS runtime
ARG TERRAFORM_VERSION=1.10.4
ARG KUBECTL_VERSION=v1.31.4

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates curl unzip openssh-client \
 && rm -rf /var/lib/apt/lists/*

# Terraform — the binary ilabhud shells out to for cloud-provider apply /
# destroy. Pinned to a known-good release for reproducibility; bump as
# needed.
RUN ARCH="$(uname -m)" \
 && case "$ARCH" in \
        x86_64)  TF_ARCH=amd64 ;; \
        aarch64) TF_ARCH=arm64 ;; \
        *) echo "unsupported arch $ARCH" >&2; exit 1 ;; \
    esac \
 && curl -fsSL "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_${TF_ARCH}.zip" -o /tmp/tf.zip \
 && unzip -j /tmp/tf.zip -d /usr/local/bin/ terraform \
 && chmod +x /usr/local/bin/terraform \
 && rm /tmp/tf.zip

# kubectl — used by the validator for kind: kubectl checks against the
# session kubeconfig.
RUN ARCH="$(uname -m)" \
 && case "$ARCH" in \
        x86_64)  K_ARCH=amd64 ;; \
        aarch64) K_ARCH=arm64 ;; \
        *) echo "unsupported arch $ARCH" >&2; exit 1 ;; \
    esac \
 && curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${K_ARCH}/kubectl" -o /usr/local/bin/kubectl \
 && chmod +x /usr/local/bin/kubectl

# Non-root user — sessions live under /var/lib/ilabhu, which is the only
# writable path the daemon needs.
RUN useradd -r -u 1000 -m -d /var/lib/ilabhu -s /bin/sh ilabhu

COPY --from=build /out/ilabhud /usr/local/bin/ilabhud

USER ilabhu
WORKDIR /var/lib/ilabhu

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/ilabhud"]
CMD ["-addr", ":8080", "-exams-dir", "/exams", "-state-dir", "/var/lib/ilabhu"]
